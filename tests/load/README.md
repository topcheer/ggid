# GGID Load Tests

## Prerequisites
```bash
npm install -g k6
# or
brew install k6
```

## Running Tests

### Login Flow (p95 < 200ms)
```bash
k6 run tests/load/login-flow.js
```

### Policy Evaluation (p95 < 50ms)
```bash
AUTH_TOKEN=$(curl -s http://localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"admin@ggid.local","password":"Admin@123456"}' | jq -r .access_token)
k6 run tests/load/policy-eval.js
```

### Risk Evaluation (p95 < 100ms)
```bash
k6 run tests/load/risk-eval.js
```

### User CRUD (p95 < 150ms)
```bash
k6 run tests/load/user-crud.js
```

## Stress Test (1000 concurrent)
```bash
k6 run --vus 1000 --duration 5m tests/load/policy-eval.js
```

## Soak Test (24h)
```bash
k6 run --vus 500 --duration 24h tests/load/login-flow.js
```

## Performance Budgets
| Endpoint | Target p95 | Max RPS |
|----------|-----------|---------|
| /auth/login | 200ms | 100 |
| /policy/authorize | 50ms | 500 |
| /policy/risk/evaluate | 100ms | 200 |
| /identity/users CRUD | 150ms | 100 |
