.PHONY: run tidy test webui-install webui-dev webui-build codex-plan

run:
	go run ./cmds server --config config.yaml

worker:
	go run ./cmds worker --config config.yaml

tidy:
	go mod tidy

test:
	go test ./...

webui-install:
	cd webui && pnpm install

webui-dev:
	cd webui && pnpm dev

webui-build:
	cd webui && pnpm build

codex-plan:
	python3 scripts/run_codex_on_plan.py --plan-file docs/plans/example.md --dry-run --max-iteration 1
