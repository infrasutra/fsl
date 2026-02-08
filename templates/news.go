package templates

var NewsPortalTemplate = TemplateDefinition{
	Name:        "News Portal",
	Slug:        "news-portal",
	Description: "Professional news website with journalists, articles, and breaking news",
	Icon:        "Newspaper",
	Category:    "content",
	Schemas: []SchemaDefinition{
		{
			Name:  "Journalist",
			ApiID: "journalist",
			Icon:  "UserCircle",
			FSL: `type Journalist {
  name: String!
  title: String
  bio: RichText
  photo: Image
  email: String!
  twitter_handle: String
  expertise: [String]
}`,
		},
		{
			Name:  "News Category",
			ApiID: "news_category",
			Icon:  "Tag",
			FSL: `type NewsCategory {
  name: String!
  slug: String! @unique
  description: String
  color: String
  parent: NewsCategory @relation
}`,
		},
		{
			Name:  "Article",
			ApiID: "article",
			Icon:  "FileText",
			FSL: `type Article {
  headline: String!
  slug: String! @unique
  subheadline: String
  content: RichText!
  featured_image: Image
  thumbnail: Image
  journalist: Journalist! @relation
  category: NewsCategory! @relation
  tags: [String]
  source: String
  is_featured: Boolean
  is_opinion: Boolean
  published_at: DateTime
  updated_at: DateTime
}`,
		},
		{
			Name:        "Breaking News",
			ApiID:       "breaking_news",
			Icon:        "Zap",
			IsSingleton: true,
			FSL: `@singleton
type BreakingNews {
  enabled: Boolean!
  headline: String!
  summary: String
  link_text: String
  link_url: String
  article: Article @relation
  expires_at: DateTime
}`,
		},
	},
	SampleDocuments: map[string][]map[string]any{
		"journalist": {
			{
				"name":  "Michael Chen",
				"title": "Senior Editor",
				"bio": []any{
					map[string]any{"type": "paragraph", "children": []any{map[string]any{"text": "Award-winning journalist covering technology and business for over 15 years."}}},
				},
				"email":          "michael.chen@example.com",
				"twitter_handle": "@mchen_news",
				"expertise":      []string{"Technology", "Business", "Startups"},
			},
			{
				"name":  "Sarah Johnson",
				"title": "Political Correspondent",
				"bio": []any{
					map[string]any{"type": "paragraph", "children": []any{map[string]any{"text": "Political analyst and correspondent with expertise in domestic policy."}}},
				},
				"email":          "sarah.johnson@example.com",
				"twitter_handle": "@sjohnson_pol",
				"expertise":      []string{"Politics", "Policy", "Government"},
			},
		},
		"news_category": {
			{
				"name":        "World News",
				"slug":        "world",
				"description": "International news and global affairs",
				"color":       "#1a73e8",
			},
			{
				"name":        "Technology",
				"slug":        "technology",
				"description": "Tech industry news and innovations",
				"color":       "#34a853",
			},
			{
				"name":        "Politics",
				"slug":        "politics",
				"description": "Political news and analysis",
				"color":       "#ea4335",
			},
		},
		"article": {
			{
				"headline":    "Welcome to Your News Portal",
				"slug":        "welcome-to-your-news-portal",
				"subheadline": "Get started with professional journalism",
				"content": []any{
					map[string]any{"type": "paragraph", "children": []any{map[string]any{"text": "This is your first article. Start publishing quality journalism with your new news portal!"}}},
				},
				"journalist":  "@ref:journalist:0",
				"category":    "@ref:news_category:1",
				"tags":        []string{"welcome", "getting-started"},
				"is_featured": true,
				"is_opinion":  false,
			},
		},
		"breaking_news": {
			{
				"enabled":   false,
				"headline":  "Breaking News Placeholder",
				"summary":   "Configure your breaking news banner here",
				"link_text": "Read More",
			},
		},
	},
}
