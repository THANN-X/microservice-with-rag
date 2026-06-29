package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"product_service/internal/config"
	repository "product_service/internal/adapter/repository/postgres"
	"product_service/internal/adapter/repository/postgres/entity"
	"product_service/internal/core/domain"

	"gorm.io/gorm"
)

type Seeder struct {
	db *gorm.DB
}

func main() {
	fmt.Println("🌱 Starting Database Seeder...")

	// 1. Load config & open database
	cfg := config.Loadconfig()
	dsn := cfg.GetDSN()
	fmt.Printf("Connecting to database: %s\n", dsn)
	
	db := config.OpenDatabase(dsn)
	seeder := &Seeder{db: db}

	// 2. Clean old data if any (Truncate tables)
	seeder.CleanAllData()

	// 3. Seed Categories, Attributes, and AttributeValues
	categories := seeder.SeedCategories()
	attributeValues := seeder.SeedAttributesAndValues()

	// 4. Seed Products
	seeder.SeedProducts(categories, attributeValues)

	fmt.Println("🎉 Database Seeding Completed Successfully!")
}

func (s *Seeder) CleanAllData() {
	fmt.Println("🗑️ Cleaning old data from tables...")
	
	// Disable triggers / constraint checks to avoid foreign key errors during truncate in Postgres
	s.db.Exec("TRUNCATE TABLE outbox_event CASCADE")
	s.db.Exec("TRUNCATE TABLE inbox_event CASCADE")
	s.db.Exec("TRUNCATE TABLE product_categories CASCADE")
	s.db.Exec("TRUNCATE TABLE variant_values CASCADE")
	s.db.Exec("TRUNCATE TABLE product_variant_entities CASCADE")
	s.db.Exec("TRUNCATE TABLE product_entities CASCADE")
	s.db.Exec("TRUNCATE TABLE category_entities CASCADE")
	s.db.Exec("TRUNCATE TABLE attribute_value_entities CASCADE")
	s.db.Exec("TRUNCATE TABLE attribute_entities CASCADE")
	
	fmt.Println("✅ Cleaning done.")
}

func (s *Seeder) SeedCategories() map[string]*domain.Category {
	fmt.Println("🏷️ Seeding categories...")

	categoriesData := []struct {
		Name        string
		Slug        string
		Description string
		ParentSlug  string
	}{
		// Root Categories
		{Name: "อุปกรณ์อิเล็กทรอนิกส์", Slug: "electronics", Description: "อุปกรณ์อิเล็กทรอนิกส์และเทคโนโลยีล้ำสมัย", ParentSlug: ""},
		{Name: "แฟชั่นและเสื้อผ้า", Slug: "fashion", Description: "เสื้อผ้าและเครื่องประดับยอดนิยม", ParentSlug: ""},
		{Name: "บ้านและห้องครัว", Slug: "home-kitchen", Description: "ของตกแต่งบ้าน อุปกรณ์ครัว และของใช้ในบ้าน", ParentSlug: ""},
		
		// Sub-categories under Electronics
		{Name: "สมาร์ทโฟน", Slug: "smartphones", Description: "โทรศัพท์มือถือและแท็บเล็ต", ParentSlug: "electronics"},
		{Name: "แล็ปท็อป", Slug: "laptops", Description: "คอมพิวเตอร์พกพาและโน้ตบุ๊ก", ParentSlug: "electronics"},
		{Name: "อุปกรณ์เสริม", Slug: "accessories", Description: "หูฟัง สายชาร์จ และอุปกรณ์เสริมไอที", ParentSlug: "electronics"},
		
		// Sub-categories under Fashion
		{Name: "เสื้อผ้าผู้ชาย", Slug: "menswear", Description: "เสื้อผ้าและรองเท้าแฟชั่นผู้ชาย", ParentSlug: "fashion"},
		{Name: "เสื้อผ้าผู้หญิง", Slug: "womenswear", Description: "เสื้อผ้าและรองเท้าแฟชั่นผู้หญิง", ParentSlug: "fashion"},
	}

	categoryMap := make(map[string]*domain.Category)

	// Phase 1: Insert root categories
	for _, cat := range categoriesData {
		if cat.ParentSlug == "" {
			c := &entity.CategoryEntity{
				Name:        cat.Name,
				Slug:        cat.Slug,
				Description: cat.Description,
				IsActive:    true,
				ParentID:    nil,
			}
			if err := s.db.Create(c).Error; err != nil {
				log.Fatalf("failed to seed root category: %v", err)
			}
			domainCat := c.ToCategoryDomain()
			categoryMap[cat.Slug] = domainCat
			fmt.Printf("Created root category: %s (ID: %d)\n", domainCat.Name, domainCat.ID)
		}
	}

	// Phase 2: Insert sub-categories
	for _, cat := range categoriesData {
		if cat.ParentSlug != "" {
			parent, ok := categoryMap[cat.ParentSlug]
			if !ok {
				log.Fatalf("parent category not found: %s", cat.ParentSlug)
			}
			parentID := parent.ID
			c := &entity.CategoryEntity{
				Name:        cat.Name,
				Slug:        cat.Slug,
				Description: cat.Description,
				IsActive:    true,
				ParentID:    &parentID,
			}
			if err := s.db.Create(c).Error; err != nil {
				log.Fatalf("failed to seed subcategory: %v", err)
			}
			domainCat := c.ToCategoryDomain()
			categoryMap[cat.Slug] = domainCat
			fmt.Printf("Created sub-category: %s (ID: %d, Parent ID: %d)\n", domainCat.Name, domainCat.ID, parentID)
		}
	}

	return categoryMap
}

func (s *Seeder) SeedAttributesAndValues() map[string][]uint {
	fmt.Println("🎨 Seeding attributes and values...")

	attributesData := map[string][]string{
		"สี":   {"แดง", "น้ำเงิน", "ดำ", "ขาว", "เทา", "เขียว"},
		"ไซส์":  {"S", "M", "L", "XL"},
		"ความจุ": {"128GB", "256GB", "512GB"},
	}

	valueIDMap := make(map[string][]uint)

	for attrName, values := range attributesData {
		attr := &entity.AttributeEntity{
			Name: attrName,
		}
		if err := s.db.Create(attr).Error; err != nil {
			log.Fatalf("failed to seed attribute %s: %v", attrName, err)
		}
		
		fmt.Printf("Created attribute: %s (ID: %d)\n", attr.Name, attr.ID)
		var valIDs []uint
		for _, val := range values {
			valEnt := &entity.AttributeValueEntity{
				AttributeID: attr.ID,
				Value:       val,
			}
			if err := s.db.Create(valEnt).Error; err != nil {
				log.Fatalf("failed to seed attribute value %s under %s: %v", val, attrName, err)
			}
			valIDs = append(valIDs, valEnt.ID)
		}
		valueIDMap[attrName] = valIDs
	}

	return valueIDMap
}

func (s *Seeder) SeedProducts(categories map[string]*domain.Category, attributeValues map[string][]uint) {
	fmt.Println("📦 Seeding 100 products...")

	rand.Seed(time.Now().UnixNano())

	// Sample data components in Thai
	brandNames := []string{"โนวา", "ซีนิท", "แอโร", "เอเปกซ์", "วอร์เท็กซ์", "ควอนตัม", "โอเมก้า", "อินฟินิตี้", "พัลส์", "ไททัน"}
	
	// Products under electronics -> smartphones
	phoneModels := []string{"โฟน X", "โฟน Z", "สมาร์ท ไลท์", "พิกเซล แมกซ์", "โน้ต อัลตร้า", "โฟลด์ V"}
	phonePics := []string{
		"https://images.unsplash.com/photo-1511707171634-5f897ff02aa9?w=500&auto=format&fit=crop",
		"https://images.unsplash.com/photo-1598327105666-5b89351aff97?w=500&auto=format&fit=crop",
		"https://images.unsplash.com/photo-1580910051074-3eb694886505?w=500&auto=format&fit=crop",
	}

	// Products under electronics -> laptops
	laptopModels := []string{"บุ๊ค โปร", "แอร์บุ๊ค", "โน้ตบุ๊ค เอ็กซ์ทรีม", "เฟล็กซ์ พีซี", "ครีเอเตอร์ โปร"}
	laptopPics := []string{
		"https://images.unsplash.com/photo-1496181130204-7552cc15f1e3?w=500&auto=format&fit=crop",
		"https://images.unsplash.com/photo-1531297484001-80022131f5a1?w=500&auto=format&fit=crop",
		"https://images.unsplash.com/photo-1484788984921-03950022c9ef?w=500&auto=format&fit=crop",
	}

	// Products under electronics -> accessories
	accessoryModels := []string{"หูฟังไร้สาย", "เครื่องชาร์จเร็ว", "พาวเวอร์แบงค์ 20k", "ยูเอสบีฮับ USB-C", "ลำโพงบลูทูธพกพา"}
	accessoryPics := []string{
		"https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=500&auto=format&fit=crop",
		"https://images.unsplash.com/photo-1572569511254-d8f925fe2cbb?w=500&auto=format&fit=crop",
		"https://images.unsplash.com/photo-1608248597279-f99d160bfcbc?w=500&auto=format&fit=crop",
	}

	// Products under fashion -> menswear or womenswear
	apparelNames := []string{"เสื้อยืดพรีเมียม", "กางเกงยีนส์ทรงสลิมฟิต", "เสื้อฮู้ดลำลอง", "กางเกงขาสั้นกีฬา", "เสื้อแจ็คเก็ตกันหนาว"}
	apparelPics := []string{
		"https://images.unsplash.com/photo-1521572267360-ee0c2909d518?w=500&auto=format&fit=crop",
		"https://images.unsplash.com/photo-1542272604-787c3835535d?w=500&auto=format&fit=crop",
		"https://images.unsplash.com/photo-1483985988355-763728e1935b?w=500&auto=format&fit=crop",
	}

	// Products under home & kitchen
	homeNames := []string{"ชุดแก้วเซรามิก", "มีดเชฟ 8 นิ้ว", "กระทะเคลือบสารกันติด", "กาต้มน้ำไฟฟ้า", "กล่องเก็บถนอมอาหาร"}
	homePics := []string{
		"https://images.unsplash.com/photo-1517256064527-09c53b2d0bc6?w=500&auto=format&fit=crop",
		"https://images.unsplash.com/photo-1599940824399-b87987ceb72a?w=500&auto=format&fit=crop",
		"https://images.unsplash.com/photo-1584269600464-37b1b58a9fe7?w=500&auto=format&fit=crop",
	}

	ctx := context.Background()

	for i := 1; i <= 100; i++ {
		var name string
		var desc string
		var catSlug string
		var imageList []string
		
		// Randomly assign category class
		catChoice := rand.Intn(5) // 0: smartphones, 1: laptops, 2: accessories, 3: menswear/womenswear, 4: home-kitchen
		
		brand := brandNames[rand.Intn(len(brandNames))]

		switch catChoice {
		case 0:
			catSlug = "smartphones"
			model := phoneModels[rand.Intn(len(phoneModels))]
			name = fmt.Sprintf("สมาร์ทโฟน %s รุ่น %s รุ่นที่ %d", brand, model, rand.Intn(5)+1)
			desc = fmt.Sprintf("สัมผัสประสบการณ์ใหม่ของสมาร์ทโฟนยุคถัดไปด้วย %s มาพร้อมกับหน่วยประมวลผลประสิทธิภาพสูง หน้าจอ AMOLED สีสันสดใส และระบบถ่ายภาพกลางคืนที่ยอดเยี่ยม", name)
			imageList = []string{phonePics[rand.Intn(len(phonePics))]}
		case 1:
			catSlug = "laptops"
			model := laptopModels[rand.Intn(len(laptopModels))]
			name = fmt.Sprintf("แล็ปท็อป %s รุ่น %s โปร", brand, model)
			desc = fmt.Sprintf("ทำงานและสร้างสรรค์ผลงานของคุณได้อย่างลื่นไหลด้วย %s ขับเคลื่อนด้วยชิปเซ็ตรุ่นล่าสุด แบตเตอรี่ใช้งานได้ยาวนาน และตัวเครื่องดีไซน์พรีเมียมบางเบา", name)
			imageList = []string{laptopPics[rand.Intn(len(laptopPics))]}
		case 2:
			catSlug = "accessories"
			model := accessoryModels[rand.Intn(len(accessoryModels))]
			name = fmt.Sprintf("%s %s", brand, model)
			desc = fmt.Sprintf("ยกระดับการใช้งานอุปกรณ์ดิจิทัลของคุณด้วย %s คุณภาพสูง ผลิตจากวัสดุที่ทนทาน ดีไซน์ทันสมัย และพกพาสะดวก", name)
			imageList = []string{accessoryPics[rand.Intn(len(accessoryPics))]}
		case 3:
			catSlug = "menswear"
			if rand.Float32() > 0.5 {
				catSlug = "womenswear"
			}
			model := apparelNames[rand.Intn(len(apparelNames))]
			name = fmt.Sprintf("%s %s", brand, model)
			desc = fmt.Sprintf("เพิ่มความมั่นใจให้กับตู้เสื้อผ้าของคุณด้วย %s ตัดเย็บจากผ้าฝ้ายผสมน้ำหนักเบาเกรดพรีเมียม สวมใส่สบาย ระบายอากาศได้ดี และทนทานต่อการซัก", name)
			imageList = []string{apparelPics[rand.Intn(len(apparelPics))]}
		default:
			catSlug = "home-kitchen"
			model := homeNames[rand.Intn(len(homeNames))]
			name = fmt.Sprintf("%s %s", brand, model)
			desc = fmt.Sprintf("เพิ่มความสะดวกสบายในการทำอาหารและจัดเตรียมอาหารในครัวของคุณด้วย %s ผลิตจากวัสดุคุณภาพดี ปลอดภัย และทนทาน", name)
			imageList = []string{homePics[rand.Intn(len(homePics))]}
		}

		// Get target category
		catObj, exists := categories[catSlug]
		if !exists {
			log.Fatalf("Category %s does not exist", catSlug)
		}

		// Prepare variants
		var variants []domain.ProductVariant
		numVariants := rand.Intn(3) + 1 // 1 to 3 variants
		
		for v := 1; v <= numVariants; v++ {
			sku := fmt.Sprintf("SKU-%04d-%d", i, v)
			var variantName string
			var price float64
			var stock int
			var attrs []domain.VariantAttribute
			
			// Decide attributes based on category
			colorIDs := attributeValues["สี"]
			chosenColorID := colorIDs[rand.Intn(len(colorIDs))]

			if catSlug == "smartphones" || catSlug == "laptops" {
				storageIDs := attributeValues["ความจุ"]
				chosenStorageID := storageIDs[rand.Intn(len(storageIDs))]

				attrs = []domain.VariantAttribute{
					{ID: chosenColorID},
					{ID: chosenStorageID},
				}
				
				price = float64((rand.Intn(400) + 100) * 100) // 10,000 to 50,000
				stock = rand.Intn(100) + 10
				variantName = fmt.Sprintf("รุ่นตัวเลือกที่ %d", v)
			} else if catSlug == "menswear" || catSlug == "womenswear" {
				sizeIDs := attributeValues["ไซส์"]
				chosenSizeID := sizeIDs[rand.Intn(len(sizeIDs))]

				attrs = []domain.VariantAttribute{
					{ID: chosenColorID},
					{ID: chosenSizeID},
				}
				
				price = float64((rand.Intn(20) + 5) * 100) // 500 to 2,500
				stock = rand.Intn(150) + 20
				variantName = fmt.Sprintf("ไซส์ตัวเลือกที่ %d", v)
			} else {
				attrs = []domain.VariantAttribute{
					{ID: chosenColorID},
				}
				price = float64((rand.Intn(30) + 3) * 100) // 300 to 3,300
				stock = rand.Intn(50) + 5
				variantName = fmt.Sprintf("สีตัวเลือกที่ %d", v)
			}

			variants = append(variants, domain.ProductVariant{
				Sku:         sku,
				NameVariant: variantName,
				Price:       price,
				Stock:       stock,
				IsActive:    true,
				ImageURLs:   imageList,
				Attributes:  attrs,
			})
		}

		// Create domain object
		domainProd, err := domain.NewProduct(
			name,
			desc,
			imageList,
			variants,
			[]domain.Category{*catObj},
		)
		if err != nil {
			log.Fatalf("failed to create product domain object %d: %v", i, err)
		}

		// Set default active
		domainProd.IsActive = true
		
		// Mark as created (raises domain events)
		domainProd.MarkAsCreated(1) // Admin user ID 1

		// Save via repo (creates product, variants, and inserts to outbox_events table)
		err = s.db.Transaction(func(tx *gorm.DB) error {
			txRepo, _ := repository.NewProductRepository(tx)
			return txRepo.CreateProduct(ctx, domainProd)
		})
		
		if err != nil {
			log.Fatalf("failed to save product %d: %v", i, err)
		}
		
		if i%10 == 0 {
			fmt.Printf("...Seeded %d/100 products\n", i)
		}
	}
}
