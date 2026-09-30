package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sectionLibrary = `enum Tone { calm, bold }

@description("Opening section with a heading and a button")
type Hero {
  heading: String! @label("Heading") @help("Shown in large type")
  tone: Tone
}

@description("Questions and answers")
type Faq {
  questions: JSON! @slices(question: Question)
}

type Question {
  question: String! @placeholder("How do I…?")
  answer: Text!
}`

func TestCompileWithLibrary(t *testing.T) {
	library, err := ParseLibrary(sectionLibrary)
	require.NoError(t, err)

	page := `type Landing {
  title: String!
  sections: JSON! @slices(hero: Hero, faq: Faq, note: Note)
}

type Note { text: Text! }`
	_, err = ParseAndCompileWithOptions(page, "Landing", "landing", false, Options{})
	assert.ErrorContains(t, err, "references unknown type 'Hero'")

	compiled, err := ParseAndCompileWithOptions(page, "Landing", "landing", false, Options{Library: library})
	require.NoError(t, err)
	byName := map[string]CompiledComponent{}
	for _, c := range compiled.Components {
		byName[c.Name] = c
	}
	require.Len(t, byName, 4)
	assert.True(t, byName["Hero"].Shared)
	assert.Equal(t, "Opening section with a heading and a button", byName["Hero"].Description)
	assert.True(t, byName["Question"].Shared, "nested types come from the library too")
	assert.False(t, byName["Note"].Shared)
	assert.Equal(t, "Heading", byName["Hero"].Fields[0].Decorators[DecLabel])
	assert.Equal(t, "Shown in large type", byName["Hero"].Fields[0].Decorators[DecHelp])
	assert.Equal(t, []CompiledEnum{{Name: "Tone", Values: []string{"calm", "bold"}}}, compiled.Enums)

	data := map[string]any{"title": "Home", "sections": []any{
		map[string]any{"type": "hero", "data": map[string]any{"heading": "Welcome"}},
		map[string]any{"type": "faq", "data": map[string]any{"questions": []any{map[string]any{"type": "question", "data": map[string]any{"question": "Why?"}}}}},
	}}
	errs := ValidateData(data, compiled)
	require.Len(t, errs, 1)
	assert.Equal(t, "sections[1].data.questions[0].data.answer", errs[0].Field)
}

func TestLibraryTypesCannotBeRedefined(t *testing.T) {
	library, err := ParseLibrary(sectionLibrary)
	require.NoError(t, err)
	page := `type Landing {
  sections: JSON! @slices(hero: Hero)
}

type Hero { title: String! }`

	_, err = ParseAndCompileWithOptions(page, "Landing", "landing", false, Options{Library: library})
	assert.ErrorContains(t, err, "type 'Hero' is already defined in the section library")

	result := ParseWithDiagnosticsAndOptions(page, Options{Library: library})
	require.False(t, result.Valid)
	require.Len(t, result.Diagnostics, 1)
	assert.Equal(t, "type 'Hero' is already defined in the section library", result.Diagnostics[0].Message)
	assert.Equal(t, 5, result.Diagnostics[0].StartLine)
}

func TestCompileLibrary(t *testing.T) {
	library, err := ParseLibrary(sectionLibrary)
	require.NoError(t, err)
	components, err := CompileLibrary(library)
	require.NoError(t, err)
	names := []string{}
	for _, c := range components {
		names = append(names, c.Name)
		assert.True(t, c.Shared)
	}
	assert.Equal(t, []string{"Faq", "Hero", "Question"}, names)
	assert.Equal(t, "Questions and answers", components[0].Description)

	_, err = ParseLibrary(`type Broken { items: JSON! @slices(item: Missing) }`)
	assert.ErrorContains(t, err, "references unknown type 'Missing'")
}

func TestFieldDisplayDecorators(t *testing.T) {
	result := ParseWithDiagnostics(`type Post {
  title: String! @label("Headline") @placeholder("A short headline") @help("Keep it under 70 characters")
}`)
	assert.True(t, result.Valid, result.Diagnostics)

	result = ParseWithDiagnostics(`type Post { title: String! @label(3) }`)
	require.False(t, result.Valid)
	assert.Contains(t, result.Diagnostics[0].Message, "@label needs a text value")
}

func TestDiffCoversSections(t *testing.T) {
	compile := func(library string) *CompiledSchema {
		lib, err := ParseLibrary(library)
		require.NoError(t, err)
		compiled, err := ParseAndCompileWithOptions("type Landing {\n  sections: JSON! @slices(hero: Hero)\n}", "Landing", "landing", false, Options{Library: lib})
		require.NoError(t, err)
		return compiled
	}
	before := compile("type Hero { heading: String!\n  subheading: Text }")

	safe := DiffSchemas(before, compile("type Hero { heading: String!\n  subheading: Text\n  image: Image }"))
	assert.False(t, safe.HasBreaking)
	require.Len(t, safe.Changes, 1)
	assert.Equal(t, "field 'Hero.image' was added", safe.Changes[0].Message)

	breaking := DiffSchemas(before, compile("type Hero { heading: String! }"))
	assert.True(t, breaking.HasBreaking)
	assert.Equal(t, "field 'Hero.subheading' was removed", breaking.Changes[0].Message)

	required := DiffSchemas(before, compile("type Hero { heading: String!\n  subheading: Text\n  cta: String! }"))
	assert.True(t, required.HasBreaking)
}

func TestLibraryReviewCases(t *testing.T) {
	library, err := ParseLibrary(sectionLibrary + "\n\nenum Unused { a, b }")
	require.NoError(t, err)
	opts := Options{Library: library}

	plain, err := ParseAndCompileWithOptions("type Plain { title: String! }", "Plain", "plain", false, opts)
	require.NoError(t, err)
	assert.Empty(t, plain.Enums, "library enums are added only for sections the page uses")

	withFaq, err := ParseAndCompileWithOptions("type Page {\n  sections: JSON @slices(faq: Faq)\n}", "Page", "page", false, opts)
	require.NoError(t, err)
	assert.Empty(t, withFaq.Enums)

	_, err = ParseAndCompileWithOptions("enum Tone { loud }\n\ntype Page {\n  sections: JSON @slices(hero: Hero)\n}", "Page", "page", false, opts)
	assert.ErrorContains(t, err, "enum 'Tone' is already defined in the section library")

	otherFile := ParseWithDiagnostics("type Card {\n  items: JSON! @slices(item: Item)\n}").Schema
	cards, err := ParseAndCompileWithOptions("type Page {\n  sections: JSON @slices(card: Card)\n}\n\ntype Item { label: String! }", "Page", "page", false, Options{Library: otherFile})
	require.NoError(t, err, "a library section may nest a type the page defines")
	for _, c := range cards.Components {
		assert.Equal(t, c.Name == "Card", c.Shared, c.Name)
	}

	result := ParseWithDiagnosticsAndOptions("// type Hero moved to the library\ntype HeroBanner { a: String }\ntype Hero { a: String }", opts)
	require.False(t, result.Valid)
	assert.Equal(t, 3, result.Diagnostics[0].StartLine)
	assert.Equal(t, 6, result.Diagnostics[0].StartColumn)
}

func TestDiffSlicesKeys(t *testing.T) {
	compile := func(slices string) *CompiledSchema {
		compiled, err := ParseAndCompile("type Page {\n  sections: JSON @slices("+slices+")\n}\n\ntype Hero { a: String }\n\ntype Banner { a: String }", "Page", "page", false)
		require.NoError(t, err)
		return compiled
	}
	both := compile("hero: Hero, banner: Banner")
	assert.True(t, DiffSchemas(both, compile("hero: Hero")).HasBreaking, "removing a key")
	assert.True(t, DiffSchemas(both, compile("hero: Hero, banner: Hero")).HasBreaking, "retargeting a key")
	assert.False(t, DiffSchemas(compile("hero: Hero"), both).HasBreaking, "adding a key")
}
