package seeds

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"
)

type ReviewSeeder struct{}

func (s *ReviewSeeder) Name() string {
	return "reviews"
}

func (s *ReviewSeeder) Priority() int {
	return 5
}

func (s *ReviewSeeder) Seed(db *sql.DB) error {
	fmt.Println("🌱 Starting reviews seeding...")

	// Получаем обычных пользователей (не админов)
	users, err := s.getRegularUsers(db)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	if len(users) == 0 {
		return fmt.Errorf("no regular users found for seeding reviews")
	}
	fmt.Printf("Found %d regular users for reviews\n", len(users))

	// Получаем все продукты
	products, err := s.getProducts(db)
	if err != nil {
		return fmt.Errorf("failed to get products: %w", err)
	}

	if len(products) == 0 {
		return fmt.Errorf("no products found for seeding reviews")
	}
	fmt.Printf("Found %d products for reviews\n", len(products))

	rand.Seed(time.Now().UnixNano())
	totalReviews := 0

	// Для каждого продукта создаем случайное количество отзывов
	for _, product := range products {
		// 0-15 отзывов на продукт
		numReviews := rand.Intn(16)
		if numReviews == 0 {
			continue
		}

		usedUsers := make(map[string]bool)
		reviewsAdded := 0

		for i := 0; i < numReviews; i++ {
			// Выбираем случайного пользователя, который еще не оставлял отзыв на этот продукт
			var user struct {
				ID   string
				Name string
			}

			attempts := 0
			maxAttempts := 30
			found := false

			for attempts < maxAttempts {
				user = users[rand.Intn(len(users))]
				if !usedUsers[user.ID] {
					found = true
					break
				}
				attempts++
			}

			if !found {
				// Если не нашли нового пользователя, пропускаем
				continue
			}

			usedUsers[user.ID] = true

			// Проверяем, что пользователь действительно существует
			var userExists bool
			err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", user.ID).Scan(&userExists)
			if err != nil || !userExists {
				continue
			}

			// Генерируем случайный рейтинг (1-5)
			rating := 1 + rand.Intn(5)

			// Генерируем комментарий
			comment := s.generateComment(rating, product.Name)

			// Случайная дата за последние 90 дней
			createdAt := time.Now().AddDate(0, 0, -rand.Intn(90))

			// Вставляем отзыв
			_, err = db.Exec(`
				INSERT INTO reviews (user_id, product_id, rating, comment, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, user.ID, product.ID, rating, comment, createdAt, createdAt)

			if err != nil {
				fmt.Printf("Failed to create review for product %s by user %s: %v\n", product.Name, user.ID, err)
				continue
			}
			reviewsAdded++
		}

		if reviewsAdded > 0 {
			fmt.Printf("Added %d reviews for product: %s\n", reviewsAdded, product.Name)
			totalReviews += reviewsAdded
		}
	}

	fmt.Printf("✅ Successfully created %d total reviews\n", totalReviews)
	return nil
}

// getRegularUsers получает только обычных пользователей (не админов)
func (s *ReviewSeeder) getRegularUsers(db *sql.DB) ([]struct {
	ID   string
	Name string
}, error) {
	// Даем время на завершение предыдущих транзакций
	time.Sleep(500 * time.Millisecond)

	// Пробуем получить пользователей с повторными попытками
	maxRetries := 3
	var users []struct {
		ID   string
		Name string
	}

	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			fmt.Printf("Retrying to get users for reviews (attempt %d)...\n", retry+1)
			time.Sleep(time.Duration(retry+1) * 500 * time.Millisecond)
		}

		rows, err := db.Query("SELECT id, name FROM users WHERE role = 'user' ORDER BY created_at")
		if err != nil {
			continue
		}
		defer rows.Close()

		users = nil // очищаем слайс
		for rows.Next() {
			var user struct {
				ID   string
				Name string
			}
			if err := rows.Scan(&user.ID, &user.Name); err != nil {
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

	if len(users) == 0 {
		return nil, fmt.Errorf("no regular users found after %d retries", maxRetries)
	}

	return users, nil
}

// getProducts получает все продукты
func (s *ReviewSeeder) getProducts(db *sql.DB) ([]struct {
	ID   string
	Name string
}, error) {
	rows, err := db.Query("SELECT id, name FROM products ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []struct {
		ID   string
		Name string
	}

	for rows.Next() {
		var product struct {
			ID   string
			Name string
		}
		if err := rows.Scan(&product.ID, &product.Name); err != nil {
			return nil, err
		}
		if product.ID != "" && product.Name != "" {
			products = append(products, product)
		}
	}

	return products, nil
}

// generateComment генерирует комментарий на основе рейтинга
func (s *ReviewSeeder) generateComment(rating int, productName string) string {
	comments := map[int][]string{
		5: {
			"Excellent product! Highly recommend it.",
			"Amazing quality and fast delivery.",
			"Perfect! Exactly what I was looking for.",
			"Outstanding product, will definitely buy again.",
			"Love it! Great value for money.",
			"Fantastic quality and great customer service.",
			"Best purchase I've made in a while!",
			"Absolutely perfect, exceeded my expectations.",
			"Wonderful product, very satisfied!",
			"Top quality, highly recommended!",
			"This product is amazing! Worth every penny.",
			"Exceeded all my expectations. 5 stars!",
			"Perfect in every way. So happy with this purchase.",
		},
		4: {
			"Very good product, minor issues but overall satisfied.",
			"Good quality, would recommend with minor reservations.",
			"Nice product, works as expected.",
			"Pretty good, meets most of my needs.",
			"Solid product, good value.",
			"Good quality, fast shipping.",
			"Works well, happy with the purchase.",
			"Nice product, minor improvements could be made.",
			"Good overall, would buy again.",
			"Quality product, satisfied with purchase.",
			"Very good, but could use some improvements.",
			"Solid product for the price. Happy with it.",
		},
		3: {
			"Average product, nothing special.",
			"Okay quality, could be better.",
			"Decent product, meets basic needs.",
			"Average experience, neither good nor bad.",
			"Fair quality, works but has room for improvement.",
			"Okay for the price, nothing exceptional.",
			"Average product, does the job.",
			"Decent quality, could be improved.",
			"Fair value, meets expectations.",
			"Average product, works as described.",
			"Nothing special, but does what it's supposed to.",
			"Mediocre quality. Expected better for the price.",
		},
		2: {
			"Below average quality, not what I expected.",
			"Disappointed with the quality.",
			"Poor build quality, doesn't last long.",
			"Not worth the money, quality issues.",
			"Below expectations, has several problems.",
			"Poor quality control, defective item.",
			"Not satisfied, quality is lacking.",
			"Disappointing purchase, wouldn't recommend.",
			"Poor value for money.",
			"Quality issues, not as described.",
			"Had high hopes but was let down.",
			"Would not buy again. Too many issues.",
		},
		1: {
			"Terrible product, complete waste of money.",
			"Worst purchase ever, avoid this product.",
			"Completely broken upon arrival.",
			"Poor quality, doesn't work at all.",
			"Waste of money, very disappointed.",
			"Defective product, terrible experience.",
			"Awful quality, would not recommend.",
			"Complete failure, avoid at all costs.",
			"Terrible experience, poor customer service.",
			"Worst product I've ever bought.",
			"Absolute garbage. Want my money back.",
			"Stay away from this product. Complete disappointment.",
		},
	}

	commentList := comments[rating]
	if len(commentList) == 0 {
		return "No comment provided."
	}

	return commentList[rand.Intn(len(commentList))]
}
