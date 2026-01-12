package main

import "product_service/internal/config"

func main() {
	cfg := config.Loadconfig()
	dsn := cfg.GetDSN()
	_ = config.OpenDatabase(dsn)
}
