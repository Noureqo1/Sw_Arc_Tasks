up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs

test-rate-limit:
	cd experiments/load-tests && cat k6-basics.js | docker run -i --network=host ghcr.io/grafana/k6 run -

test-throttle:
	cd experiments/load-tests && cat k6-throttling.js | docker run -i --network=host ghcr.io/grafana/k6 run -




restart:
	make down
	make up
