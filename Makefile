# Makefile - shortcut command untuk project SKP
# Cara pakai: make <command>

# Load .env file agar variabel tersedia di Makefile
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: help dev prod down logs ps clean

# ─────────────────────────────────────────
# Development
# ─────────────────────────────────────────

dev:			## Jalankan semua service dalam mode development (hot reload)
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build

dev-bg:			## Jalankan development di background
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build -d

# ─────────────────────────────────────────
# Production
# ─────────────────────────────────────────

prod:			## Jalankan semua service dalam mode production
	docker compose up --build -d

prod-build:		## Build ulang image tanpa start
	docker compose build --no-cache

# ─────────────────────────────────────────
# Utility
# ─────────────────────────────────────────

down:			## Stop dan hapus semua container (data tetap aman)
	docker compose down

down-all:		## Stop container + hapus volumes (DATA HILANG!)
	docker compose down -v

logs:			## Lihat log semua service
	docker compose logs -f

logs-backend:		## Lihat log backend saja
	docker compose logs -f backend

logs-db:		## Lihat log postgres saja
	docker compose logs -f postgres

ps:			## Lihat status semua container
	docker compose ps

restart-backend:	## Restart backend saja (tanpa rebuild)
	docker compose restart backend

shell-db:		## Masuk ke psql di container postgres
	docker compose exec postgres psql -U $(DB_USER) -d $(DB_NAME)

shell-backend:		## Masuk ke shell container backend
	docker compose exec backend sh

clean:			## Hapus image yang tidak terpakai
	docker image prune -f

help:			## Tampilkan bantuan
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'