package templates

var PortfolioTemplate = TemplateDefinition{
	Name:        "Portfolio",
	Slug:        "portfolio",
	Description: "Personal portfolio with projects, skills, and testimonials",
	Icon:        "Briefcase",
	Category:    "content",
	Schemas: []SchemaDefinition{
		{
			Name:  "Project",
			ApiID: "project",
			Icon:  "Layers",
			FSL: `type Project {
  title: String!
  slug: String! @unique
  description: RichText!
  short_description: String
  featured_image: Image
  images: [Image]
  client: String
  project_url: String
  repository_url: String
  technologies: [String]
  category: String
  start_date: Date
  end_date: Date
  is_featured: Boolean
  display_order: Int
}`,
		},
		{
			Name:  "Skill",
			ApiID: "skill",
			Icon:  "Zap",
			FSL: `type Skill {
  name: String!
  category: String!
  proficiency: Int
  icon: String
  years_experience: Int
  description: String
  display_order: Int
}`,
		},
		{
			Name:  "Testimonial",
			ApiID: "testimonial",
			Icon:  "MessageSquare",
			FSL: `type Testimonial {
  author_name: String!
  author_title: String
  author_company: String
  author_photo: Image
  content: String!
  rating: Int
  project: Project @relation
  is_featured: Boolean
  display_order: Int
}`,
		},
		{
			Name:        "About",
			ApiID:       "about",
			Icon:        "User",
			IsSingleton: true,
			FSL: `@singleton
type About {
  name: String!
  title: String!
  tagline: String
  bio: RichText!
  short_bio: String
  photo: Image
  resume: File
  email: String
  phone: String
  location: String
  available_for_hire: Boolean
  social_links: JSON
}`,
		},
	},
	SampleDocuments: map[string][]map[string]any{
		"project": {
			{
				"title": "E-commerce Platform Redesign",
				"slug":  "ecommerce-platform-redesign",
				"description": []any{
					map[string]any{"type": "paragraph", "children": []any{map[string]any{"text": "Complete redesign of a major e-commerce platform, improving UX and increasing conversion rates by 40%."}}},
				},
				"short_description": "UX redesign that boosted conversions by 40%",
				"client":            "TechRetail Inc.",
				"project_url":       "https://example.com/project1",
				"technologies":      []string{"React", "TypeScript", "Node.js", "PostgreSQL"},
				"category":          "Web Development",
				"is_featured":       true,
				"display_order":     1,
			},
			{
				"title": "Mobile Banking App",
				"slug":  "mobile-banking-app",
				"description": []any{
					map[string]any{"type": "paragraph", "children": []any{map[string]any{"text": "Designed and developed a secure mobile banking application with biometric authentication and real-time transactions."}}},
				},
				"short_description": "Secure mobile banking with biometric auth",
				"client":            "FinanceFirst Bank",
				"technologies":      []string{"React Native", "TypeScript", "GraphQL"},
				"category":          "Mobile Development",
				"is_featured":       true,
				"display_order":     2,
			},
		},
		"skill": {
			{
				"name":             "React",
				"category":         "Frontend",
				"proficiency":      95,
				"icon":             "react",
				"years_experience": 6,
				"description":      "Building complex SPAs and component libraries",
				"display_order":    1,
			},
			{
				"name":             "TypeScript",
				"category":         "Languages",
				"proficiency":      90,
				"icon":             "typescript",
				"years_experience": 5,
				"description":      "Type-safe development for large-scale applications",
				"display_order":    2,
			},
			{
				"name":             "Node.js",
				"category":         "Backend",
				"proficiency":      85,
				"icon":             "nodejs",
				"years_experience": 7,
				"description":      "RESTful APIs and microservices architecture",
				"display_order":    3,
			},
			{
				"name":             "UI/UX Design",
				"category":         "Design",
				"proficiency":      80,
				"icon":             "figma",
				"years_experience": 4,
				"description":      "User-centered design and prototyping",
				"display_order":    4,
			},
		},
		"testimonial": {
			{
				"author_name":    "Emily Rodriguez",
				"author_title":   "CTO",
				"author_company": "TechRetail Inc.",
				"content":        "Exceptional work on our platform redesign. The attention to detail and technical expertise resulted in a 40% increase in our conversion rates.",
				"rating":         5,
				"project":        "@ref:project:0",
				"is_featured":    true,
				"display_order":  1,
			},
			{
				"author_name":    "David Park",
				"author_title":   "Product Manager",
				"author_company": "FinanceFirst Bank",
				"content":        "Delivered a secure, user-friendly mobile banking experience that our customers love. Professional and reliable throughout the project.",
				"rating":         5,
				"project":        "@ref:project:1",
				"is_featured":    true,
				"display_order":  2,
			},
		},
		"about": {
			{
				"name":    "Your Name",
				"title":   "Full Stack Developer",
				"tagline": "Building digital experiences that matter",
				"bio": []any{
					map[string]any{"type": "paragraph", "children": []any{map[string]any{"text": "I'm a passionate full-stack developer with 8+ years of experience creating web and mobile applications. I specialize in React, TypeScript, and Node.js, with a strong focus on user experience and clean code."}}},
					map[string]any{"type": "paragraph", "children": []any{map[string]any{"text": "When I'm not coding, you'll find me exploring new technologies, contributing to open source, or sharing knowledge through tech blogs and meetups."}}},
				},
				"short_bio":          "Full-stack developer passionate about creating impactful digital experiences",
				"email":              "hello@example.com",
				"location":           "San Francisco, CA",
				"available_for_hire": true,
				"social_links": map[string]string{
					"github":   "https://github.com/username",
					"linkedin": "https://linkedin.com/in/username",
					"twitter":  "https://twitter.com/username",
				},
			},
		},
	},
}
