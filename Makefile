# Сборка и деплой сервисов RealTimeMap.
#
# Прод (2 ядра / 4 ГБ) образы не собирает — только тянет из GHCR.
# Сборка идёт в GitHub Actions либо локально на машине разработчика.

SERVICES := mark-service comment-service feedback-service gamification-service smtp-service social-service
REGISTRY := ghcr.io
OWNER    := realtimemap
TAG      ?= latest

# Ограничители нагрузки для слабых машин (см. build/Dockerfile)
JOBS ?=
PROCS ?=
BUILD_ARGS := $(if $(JOBS),--build-arg BUILD_JOBS=$(JOBS)) $(if $(PROCS),--build-arg GOMAXPROCS=$(PROCS))

.PHONY: help pull up down deploy logs ps build build-one push-one

help:
	@echo "Прод:"
	@echo "  make deploy            — pull свежих образов + перезапуск"
	@echo "  make pull / up / down  — по отдельности"
	@echo "  make ps / logs         — состояние и логи"
	@echo ""
	@echo "Локальная сборка (не на проде):"
	@echo "  make build                       — все сервисы"
	@echo "  make build-one S=mark-service    — один сервис"
	@echo "  make push-one  S=mark-service    — собрать и запушить в GHCR"
	@echo ""
	@echo "Щадящий режим на слабой машине:"
	@echo "  make build-one S=mark-service JOBS=1 PROCS=1"

# ---------- прод ----------

pull:
	docker compose pull

up:
	docker compose up -d

down:
	docker compose down

deploy: pull up
	@echo "Готово. Состояние:"
	@docker compose ps

ps:
	docker compose ps

logs:
	docker compose logs -f --tail=100

# ---------- локальная сборка ----------

build:
	@for s in $(SERVICES); do \
		echo "=== $$s ==="; \
		docker build $(BUILD_ARGS) --build-arg SERVICE=$$s \
			-f build/Dockerfile -t $(REGISTRY)/$(OWNER)/$$s:$(TAG) . || exit 1; \
	done

build-one:
	@test -n "$(S)" || { echo "Укажи сервис: make build-one S=mark-service"; exit 1; }
	docker build $(BUILD_ARGS) --build-arg SERVICE=$(S) \
		-f build/Dockerfile -t $(REGISTRY)/$(OWNER)/$(S):$(TAG) .

push-one: build-one
	docker push $(REGISTRY)/$(OWNER)/$(S):$(TAG)
