#!/bin/bash

# =============================================================================
# Script de Pruebas cURL - API de Transacciones
# Ejecuta de manera automatizada los casos de éxito y de error simulado (Mock)
# =============================================================================

API_URL="http://localhost:8000/api/v1/transactions"

echo "=========================================================="
echo " Ejecutando Pruebas End-to-End del API de Transacciones"
echo "=========================================================="
echo ""

# 1. Caso de Éxito
echo "➡️  Prueba 1: Caso de Éxito (Transacción Aprobada)"
curl -sS -i -X POST \
  -H "Content-Type: application/json" \
  -d '{
  "accountId": "acc-123456",
  "type": "CREDIT",
  "amount": 1500.00,
  "currency": "MXN",
  "description": "Transferencia recibida (Éxito)"
}' $API_URL
echo -e "\n\n"

# 2. Caso de Error: Fondos Insuficientes
echo "➡️  Prueba 2: Simulación de Fondos Insuficientes (X-Mock-Id: id-insufficient-funds)"
curl -sS -i -X POST \
  -H "Content-Type: application/json" \
  -H "X-Mock-Id: id-insufficient-funds" \
  -d '{
  "accountId": "acc-123456",
  "type": "DEBIT",
  "amount": 50000.00,
  "currency": "MXN",
  "description": "Compra de auto de lujo (Rechazada)"
}' $API_URL
echo -e "\n\n"

# 3. Caso de Error: Caída del Proveedor 500
echo "➡️  Prueba 3: Simulación de Caída del Proveedor (X-Mock-Id: id-error-500)"
curl -sS -i -X POST \
  -H "Content-Type: application/json" \
  -H "X-Mock-Id: id-error-500" \
  -d '{
  "accountId": "acc-123456",
  "type": "CREDIT",
  "amount": 100.00,
  "currency": "MXN",
  "description": "Prueba de crash del proveedor"
}' $API_URL
echo -e "\n\n"

# 4. Caso de Error: Mock Inválido
echo "➡️  Prueba 4: Simulación de Mock ID Desconocido (X-Mock-Id: id-desconocido)"
curl -sS -i -X POST \
  -H "Content-Type: application/json" \
  -H "X-Mock-Id: id-desconocido" \
  -d '{
  "accountId": "acc-123456",
  "type": "CREDIT",
  "amount": 250.00,
  "currency": "MXN",
  "description": "Prueba de header inválido"
}' $API_URL
echo -e "\n\n"

# =============================================================================
# GET /transactions - Consulta de Transacciones
# =============================================================================
echo "=========================================================="
echo " Consulta de Transacciones (GET /transactions)"
echo "=========================================================="
echo ""

# 5. GET sin filtros (lista todas las transacciones)
echo "➡️  Prueba 5: GET /transactions sin filtros (paginación por defecto)"
curl -sS -i -X GET "${API_URL}"
echo -e "\n\n"

# 6. GET filtrado por accountId
echo "➡️  Prueba 6: GET /transactions filtrado por accountId"
curl -sS -i -X GET "${API_URL}?accountId=acc-123456"
echo -e "\n\n"

# 7. GET filtrado por status
echo "➡️  Prueba 7: GET /transactions filtrado por status=EXECUTED"
curl -sS -i -X GET "${API_URL}?status=EXECUTED"
echo -e "\n\n"

# 8. GET filtrado por type
echo "➡️  Prueba 8: GET /transactions filtrado por type=CREDIT"
curl -sS -i -X GET "${API_URL}?type=CREDIT"
echo -e "\n\n"

# 9. GET con paginación
echo "➡️  Prueba 9: GET /transactions con paginación (page=1, limit=5)"
curl -sS -i -X GET "${API_URL}?page=1&limit=5"
echo -e "\n\n"

# 10. GET con todos los filtros combinados
echo "➡️  Prueba 10: GET /transactions con todos los filtros combinados"
curl -sS -i -X GET "${API_URL}?accountId=acc-123456&status=EXECUTED&type=CREDIT&page=1&limit=10"
echo -e "\n\n"

# 11. GET con Cache-Control: no-cache (bypass de caché)
echo "➡️  Prueba 11: GET /transactions con Cache-Control: no-cache (bypass caché)"
curl -sS -i -X GET \
  -H "Cache-Control: no-cache" \
  "${API_URL}?accountId=acc-123456"
echo -e "\n\n"

# 12. GET con page inválido (debe retornar 400)
echo "➡️  Prueba 12: GET /transactions con page inválido (error 400 esperado)"
curl -sS -i -X GET "${API_URL}?page=abc"
echo -e "\n\n"

# 13. GET con limit inválido (debe retornar 400)
echo "➡️  Prueba 13: GET /transactions con limit inválido (error 400 esperado)"
curl -sS -i -X GET "${API_URL}?limit=xyz"
echo -e "\n\n"

# 14. GET que retorna array vacio (cuenta inexistente o sin transacciones)
echo "➡️  Prueba 14: GET /transactions con cuenta sin transacciones (array vacio esperado)"
curl -sS -i -X GET "${API_URL}?accountId=acc-empty-123456"
echo -e "\n\n"

echo "=========================================================="
echo " ✅ Pruebas cURL finalizadas."
echo "=========================================================="
