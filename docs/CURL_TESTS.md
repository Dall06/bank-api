# Pruebas de la API de Transacciones (cURL)

Este archivo contiene la colección de comandos `curl` para probar todos los escenarios del microservicio de transacciones a través del API Gateway (`http://localhost:8000`).

Como la autenticación JWT (`AUTH_REQUIRED`) está deshabilitada para pruebas, no necesitas enviar el header `Authorization`.

## 1. Caso de Éxito (Flujo Normal)
Este es el escenario "Happy Path". El proveedor externo mockeado aprobará la transacción y devolverá un saldo actualizado.

```bash
curl -sS -i -X POST \
  -H "Content-Type: application/json" \
  -d '{
  "accountId": "acc-123456",
  "type": "CREDIT",
  "amount": 1500.00,
  "currency": "MXN",
  "description": "Transferencia recibida"
}' http://localhost:8000/api/v1/transactions
```

---

## 2. Caso de Error: Fondos Insuficientes (`INSUFFICIENT_FUNDS`)
Se utiliza el header `X-Mock-Id: id-insufficient-funds`. El proveedor rechazará la operación indicando que la cuenta no tiene fondos. La base de datos persistirá la transacción con estado `REJECTED`.

```bash
curl -sS -i -X POST \
  -H "Content-Type: application/json" \
  -H "X-Mock-Id: id-insufficient-funds" \
  -d '{
  "accountId": "acc-123456",
  "type": "DEBIT",
  "amount": 50000.00,
  "currency": "MXN",
  "description": "Compra de auto de lujo"
}' http://localhost:8000/api/v1/transactions
```

---

## 3. Caso de Error: Falla Interna del Proveedor (`INTERNAL_PROVIDER_ERROR`)
Se utiliza el header `X-Mock-Id: id-error-500`. Simula una caída total del proveedor bancario externo (un HTTP 500). El sistema maneja el error gracefully y marca la transacción como `REJECTED`.

```bash
curl -sS -i -X POST \
  -H "Content-Type: application/json" \
  -H "X-Mock-Id: id-error-500" \
  -d '{
  "accountId": "acc-123456",
  "type": "CREDIT",
  "amount": 100.00,
  "currency": "MXN",
  "description": "Prueba de crash del proveedor"
}' http://localhost:8000/api/v1/transactions
```

---

## 4. Caso de Error: Mock ID Inválido (`INVALID_MOCK_ID`)
Se utiliza un ID de mock que no existe en el catálogo del proveedor (ej. `id-desconocido`). El proveedor responde con Bad Request.

```bash
curl -sS -i -X POST \
  -H "Content-Type: application/json" \
  -H "X-Mock-Id: id-desconocido" \
  -d '{
  "accountId": "acc-123456",
  "type": "CREDIT",
  "amount": 250.00,
  "currency": "MXN",
  "description": "Prueba de header inválido"
}' http://localhost:8000/api/v1/transactions
```
