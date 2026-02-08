package templates

var BlogTemplate = TemplateDefinition{
	Name:        "Blog",
	Slug:        "blog",
	Description: "Complete blog system with posts, authors, and categories",
	Icon:        "BookOpen",
	Category:    "content",
	Schemas: []SchemaDefinition{
		{
			Name:  "Author",
			ApiID: "author",
			Icon:  "User",
			FSL: `type Author {
  name: String!
  bio: RichText
  avatar: Image
  email: String
  social_links: JSON
}`,
		},
		{
			Name:  "Category",
			ApiID: "category",
			Icon:  "Folder",
			FSL: `type Category {
  name: String!
  slug: String! @unique
  description: String
}`,
		},
		{
			Name:  "Blog Post",
			ApiID: "blog_post",
			Icon:  "FileText",
			FSL: `type BlogPost {
  title: String!
  slug: String! @unique
  content: RichText!
  excerpt: String
  featured_image: Image
  author: Author! @relation
  categories: [Category] @relation
  tags: [String]
  published_at: DateTime
}`,
		},
	},
	SampleDocuments: map[string][]map[string]any{
		"author": {
			{
				"name": "Jane Smith",
				"bio": []any{
					map[string]any{"type": "paragraph", "children": []any{map[string]any{"text": "Senior content writer with 10 years of experience"}}},
				},
				"email": "jane@example.com",
			},
		},
		"category": {
			{
				"name":        "Technology",
				"slug":        "technology",
				"description": "Tech news and tutorials",
			},
			{
				"name":        "Lifestyle",
				"slug":        "lifestyle",
				"description": "Life tips and experiences",
			},
		},
		"blog_post": {
			{
				"title": "Welcome to Our Blog",
				"slug":  "welcome-to-our-blog",
				"content": []any{
					map[string]any{"type": "paragraph", "children": []any{map[string]any{"text": "This is your first blog post. Edit this content to get started with your new blog!"}}},
				},
				"excerpt":    "Get started with your new blog",
				"author":     "@ref:author:0",
				"categories": []string{"@ref:category:0"},
				"tags":       []string{"welcome", "getting-started"},
			},
		},
	},
}
