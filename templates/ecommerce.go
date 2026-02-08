package templates

var EcommerceTemplate = TemplateDefinition{
	Name:        "E-commerce",
	Slug:        "ecommerce",
	Description: "Product catalog with brands, categories, and reviews",
	Icon:        "ShoppingCart",
	Category:    "commerce",
	Schemas: []SchemaDefinition{
		{
			Name:  "Brand",
			ApiID: "brand",
			Icon:  "Award",
			FSL: `type Brand {
  name: String!
  slug: String! @unique
  description: RichText
  logo: Image
  website: String
  featured: Boolean
}`,
		},
		{
			Name:  "Product Category",
			ApiID: "product_category",
			Icon:  "Grid",
			FSL: `type ProductCategory {
  name: String!
  slug: String! @unique
  description: String
  image: Image
  parent: ProductCategory @relation
  display_order: Int
}`,
		},
		{
			Name:  "Product",
			ApiID: "product",
			Icon:  "Package",
			FSL: `type Product {
  name: String!
  slug: String! @unique
  sku: String! @unique
  description: RichText!
  short_description: String
  price: Float!
  compare_at_price: Float
  cost_price: Float
  brand: Brand @relation
  category: ProductCategory! @relation
  images: [Image]
  thumbnail: Image
  weight: Float
  dimensions: JSON
  tags: [String]
  is_active: Boolean
  is_featured: Boolean
  inventory_quantity: Int
  low_stock_threshold: Int
  meta_title: String
  meta_description: String
}`,
		},
		{
			Name:  "Product Review",
			ApiID: "product_review",
			Icon:  "Star",
			FSL: `type ProductReview {
  product: Product! @relation
  reviewer_name: String!
  reviewer_email: String!
  rating: Int!
  title: String
  content: String!
  pros: [String]
  cons: [String]
  verified_purchase: Boolean
  is_approved: Boolean
  helpful_count: Int
  reported: Boolean
}`,
		},
	},
	SampleDocuments: map[string][]map[string]any{
		"brand": {
			{
				"name": "Acme Corp",
				"slug": "acme-corp",
				"description": []any{
					map[string]any{"type": "paragraph", "children": []any{map[string]any{"text": "Quality products since 1990. Trusted by millions worldwide."}}},
				},
				"featured": true,
			},
			{
				"name": "TechGear",
				"slug": "techgear",
				"description": []any{
					map[string]any{"type": "paragraph", "children": []any{map[string]any{"text": "Innovative technology accessories for modern life."}}},
				},
				"featured": true,
			},
		},
		"product_category": {
			{
				"name":          "Electronics",
				"slug":          "electronics",
				"description":   "Gadgets, devices, and accessories",
				"display_order": 1,
			},
			{
				"name":          "Accessories",
				"slug":          "accessories",
				"description":   "Complement your devices",
				"display_order": 2,
			},
			{
				"name":          "Home & Living",
				"slug":          "home-living",
				"description":   "Products for your home",
				"display_order": 3,
			},
		},
		"product": {
			{
				"name": "Wireless Bluetooth Headphones",
				"slug": "wireless-bluetooth-headphones",
				"sku":  "WBH-001",
				"description": []any{
					map[string]any{"type": "paragraph", "children": []any{map[string]any{"text": "Premium wireless headphones with active noise cancellation. Enjoy crystal-clear audio with up to 30 hours of battery life."}}},
				},
				"short_description":   "Premium ANC headphones with 30hr battery",
				"price":               149.99,
				"compare_at_price":    199.99,
				"brand":               "@ref:brand:1",
				"category":            "@ref:product_category:0",
				"tags":                []string{"wireless", "bluetooth", "headphones", "audio"},
				"is_active":           true,
				"is_featured":         true,
				"inventory_quantity":  100,
				"low_stock_threshold": 10,
			},
			{
				"name": "USB-C Charging Cable",
				"slug": "usb-c-charging-cable",
				"sku":  "UCC-001",
				"description": []any{
					map[string]any{"type": "paragraph", "children": []any{map[string]any{"text": "Durable braided USB-C cable with fast charging support. Compatible with all USB-C devices."}}},
				},
				"short_description":   "Fast-charging braided USB-C cable",
				"price":               19.99,
				"brand":               "@ref:brand:0",
				"category":            "@ref:product_category:1",
				"tags":                []string{"cable", "usb-c", "charging"},
				"is_active":           true,
				"is_featured":         false,
				"inventory_quantity":  500,
				"low_stock_threshold": 50,
			},
		},
		"product_review": {
			{
				"product":           "@ref:product:0",
				"reviewer_name":     "Alex Thompson",
				"reviewer_email":    "alex@example.com",
				"rating":            5,
				"title":             "Best headphones I've owned!",
				"content":           "Amazing sound quality and the noise cancellation is top-notch. Battery lasts forever!",
				"pros":              []string{"Great sound", "Long battery", "Comfortable"},
				"cons":              []string{"Slightly heavy"},
				"verified_purchase": true,
				"is_approved":       true,
				"helpful_count":     12,
			},
		},
	},
}
