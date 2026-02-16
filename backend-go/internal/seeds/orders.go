package seeds

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"
)

type OrderSeeder struct{}

func (s *OrderSeeder) Name() string {
	return "orders"
}

func (s *OrderSeeder) Priority() int {
	return 4
}

func (s *OrderSeeder) Seed(db *sql.DB) error {
	fmt.Println("🌱 Starting orders seeding...")

	// Получаем обычных пользователей (не админов)
	users, err := s.getRegularUsers(db)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	if len(users) == 0 {
		return fmt.Errorf("no regular users found for seeding orders")
	}
	fmt.Printf("Found %d regular users for orders\n", len(users))

	// Получаем продукты
	products, err := s.getProducts(db)
	if err != nil {
		return fmt.Errorf("failed to get products: %w", err)
	}

	if len(products) == 0 {
		return fmt.Errorf("no products found for seeding orders")
	}
	fmt.Printf("Found %d products for orders\n", len(products))

	rand.Seed(time.Now().UnixNano())
	orderStatuses := []string{"pending", "processing", "shipped", "delivered", "cancelled"}

	// Создаем 50-100 заказов
	numOrders := 50 + rand.Intn(51)
	fmt.Printf("Creating %d orders...\n", numOrders)

	successCount := 0
	for i := 0; i < numOrders; i++ {
		// Выбираем случайного пользователя
		user := users[rand.Intn(len(users))]

		// Проверяем, что пользователь существует в БД
		var userExists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND role = 'user')", user.ID).Scan(&userExists)
		if err != nil || !userExists {
			fmt.Printf("User %s does not exist or is not regular user, skipping\n", user.ID)
			continue
		}

		numItems := 1 + rand.Intn(5) // 1-5 товаров в заказе
		status := orderStatuses[rand.Intn(len(orderStatuses))]
		createdAt := time.Now().AddDate(0, 0, -rand.Intn(180)) // случайная дата за последние 180 дней

		// Создаем заказ - УБИРАЕМ subtotal!
		var orderID string
		total := 0.0

		err = db.QueryRow(`
			INSERT INTO orders (user_id, total, status, shipping_address, billing_address, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`, user.ID, 0.0, status, s.generateAddress(), s.generateAddress(), createdAt, createdAt).Scan(&orderID)

		if err != nil {
			fmt.Printf("Failed to create order for user %s: %v\n", user.ID, err)
			continue
		}

		// Добавляем товары в заказ
		usedProducts := make(map[string]bool)
		itemsAdded := 0

		for j := 0; j < numItems; j++ {
			// Выбираем случайный продукт, который еще не использовали в этом заказе
			var product struct {
				ID    string
				Price float64
			}

			attempts := 0
			maxAttempts := 20
			found := false

			for attempts < maxAttempts {
				product = products[rand.Intn(len(products))]
				if !usedProducts[product.ID] {
					found = true
					break
				}
				attempts++
			}

			if !found {
				// Если не нашли новый продукт, берем любой
				product = products[rand.Intn(len(products))]
			}

			usedProducts[product.ID] = true

			quantity := 1 + rand.Intn(3) // 1-3 штуки
			itemTotal := product.Price * float64(quantity)
			total += itemTotal

			_, err = db.Exec(`
				INSERT INTO order_items (order_id, product_id, quantity, price, created_at)
				VALUES ($1, $2, $3, $4, $5)
			`, orderID, product.ID, quantity, product.Price, createdAt)

			if err != nil {
				fmt.Printf("Failed to create order item for order %s: %v\n", orderID, err)
				continue
			}
			itemsAdded++
		}

		// Обновляем общую сумму заказа
		if itemsAdded > 0 {
			_, err = db.Exec(`
				UPDATE orders SET total = $1 WHERE id = $2
			`, total, orderID)

			if err != nil {
				fmt.Printf("Failed to update order total for order %s: %v\n", orderID, err)
				continue
			}
			successCount++
		} else {
			// Если не добавили ни одного товара, удаляем заказ
			db.Exec("DELETE FROM orders WHERE id = $1", orderID)
		}
	}

	fmt.Printf("✅ Successfully created %d orders\n", successCount)
	return nil
}

// getRegularUsers получает только обычных пользователей (не админов)
func (s *OrderSeeder) getRegularUsers(db *sql.DB) ([]struct {
	ID string
}, error) {
	// Даем время на завершение предыдущих транзакций
	time.Sleep(500 * time.Millisecond)

	// Пробуем получить пользователей с повторными попытками
	maxRetries := 3
	var users []struct{ ID string }
	var err error

	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			fmt.Printf("Retrying to get users (attempt %d)...\n", retry+1)
			time.Sleep(time.Duration(retry+1) * 500 * time.Millisecond)
		}

		rows, err := db.Query("SELECT id FROM users WHERE role = 'user' ORDER BY created_at")
		if err != nil {
			continue
		}
		defer rows.Close()

		users = nil // очищаем слайс
		for rows.Next() {
			var user struct {
				ID string
			}
			if err := rows.Scan(&user.ID); err != nil {
				continue
			}
			if user.ID != "" {
				// Дополнительно проверяем, что пользователь действительно существует
				var exists bool
				checkErr := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", user.ID).Scan(&exists)
				if checkErr == nil && exists {
					users = append(users, user)
				}
			}
		}

		if len(users) > 0 {
			break
		}
	}

	return users, err
}

// getProducts получает все доступные продукты
func (s *OrderSeeder) getProducts(db *sql.DB) ([]struct {
	ID    string
	Price float64
}, error) {
	rows, err := db.Query("SELECT id, price FROM products WHERE in_stock = true")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []struct {
		ID    string
		Price float64
	}

	for rows.Next() {
		var product struct {
			ID    string
			Price float64
		}
		if err := rows.Scan(&product.ID, &product.Price); err != nil {
			return nil, err
		}
		if product.ID != "" && product.Price > 0 {
			products = append(products, product)
		}
	}

	return products, nil
}

// generateAddress генерирует случайный адрес в JSON формате
func (s *OrderSeeder) generateAddress() string {
	addresses := []string{
		`{"street": "123 Main St", "city": "New York", "state": "NY", "zip": "10001", "country": "USA"}`,
		`{"street": "456 Oak Ave", "city": "Los Angeles", "state": "CA", "zip": "90210", "country": "USA"}`,
		`{"street": "789 Pine Rd", "city": "Chicago", "state": "IL", "zip": "60601", "country": "USA"}`,
		`{"street": "321 Elm St", "city": "Houston", "state": "TX", "zip": "77001", "country": "USA"}`,
		`{"street": "654 Maple Dr", "city": "Phoenix", "state": "AZ", "zip": "85001", "country": "USA"}`,
		`{"street": "987 Cedar Ln", "city": "Philadelphia", "state": "PA", "zip": "19101", "country": "USA"}`,
		`{"street": "147 Birch Way", "city": "San Antonio", "state": "TX", "zip": "78201", "country": "USA"}`,
		`{"street": "258 Spruce St", "city": "San Diego", "state": "CA", "zip": "92101", "country": "USA"}`,
		`{"street": "369 Willow Ave", "city": "Dallas", "state": "TX", "zip": "75201", "country": "USA"}`,
		`{"street": "741 Poplar Rd", "city": "San Jose", "state": "CA", "zip": "95101", "country": "USA"}`,
		`{"street": "852 Beach Blvd", "city": "Miami", "state": "FL", "zip": "33101", "country": "USA"}`,
		`{"street": "963 Mountain Rd", "city": "Denver", "state": "CO", "zip": "80201", "country": "USA"}`,
		`{"street": "159 Lake Dr", "city": "Seattle", "state": "WA", "zip": "98101", "country": "USA"}`,
		`{"street": "753 Park Ave", "city": "Boston", "state": "MA", "zip": "02101", "country": "USA"}`,
		`{"street": "951 Broadway", "city": "Nashville", "state": "TN", "zip": "37201", "country": "USA"}`,
	}

	return addresses[rand.Intn(len(addresses))]
}
