package compose_test

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/so-heil/wishlist/foundation/compose"
)

func ExampleNew() {
	const services = `
services:
  db:
    image: postgres
    ports:
      - 5432:5432
    environment:
      POSTGRES_PASSWORD: postgres
    healthcheck:
      test: ["CMD-SHELL", "pg_isready", "-d", "db_prod"]
      interval: 300ms
      timeout: 60s
      retries: 5
      start_period: 80s  
`

	cps, err := compose.New(services)
	if err != nil {
		fmt.Printf("create services: %s\n", err)
		return
	}

	defer func(cps *compose.Compose) {
		err := cps.Close()
		if err != nil {
			fmt.Printf("close comose: %s\n", err)
		}
	}(cps)

	containers, err := cps.Up()
	if err != nil {
		fmt.Printf("start containers: %s\n", err)
		return
	}

	dbContainer := containers["db"]
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := dbContainer.WaitForHealthy(ctx); err != nil {
		fmt.Printf("wait for: %s\n", err)
		return
	}

	db, err := sql.Open("postgres", fmt.Sprintf("postgres://postgres:postgres@%s?sslmode=disable", dbContainer.Host))
	if err != nil {
		fmt.Printf("connect to db: %s\n", err)
		return
	}

	if pingErr := db.Ping(); pingErr != nil {
		fmt.Printf("ping db: %s\n", pingErr)
		return
	}

	var res bool
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.QueryRowContext(ctx, `SELECT true`).Scan(&res); err != nil {
		fmt.Printf("query db: %s\n", err)
		return
	}

	fmt.Println(res)
	// Output: true
}
