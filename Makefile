.PHONY: start stop restart logs reload status

start:
	docker compose up -d

stop:
	docker compose down

restart:
	docker compose restart

logs:
	docker compose logs -f

reload:
	docker exec minecraft-bedrock send-command "reload"

status:
	docker compose ps
