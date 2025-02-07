up:
	docker compose up -d

up_redis:
	docker compose up -d redis

down:
	docker compose down --remove-orphans --volumes

cli:
	docker compose exec -it pg psql -U postgres -d postgres