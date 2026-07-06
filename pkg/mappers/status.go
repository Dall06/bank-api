package mappers

// MapProviderStatus convierte el estado que retorna el proveedor externo
// al estado interno de nuestro sistema.
// APPROVED (proveedor) -> EXECUTED (nuestro sistema)
// Cualquier otro valor -> REJECTED
func MapProviderStatus(providerStatus string) string {
	if providerStatus == "APPROVED" {
		return "EXECUTED"
	}
	return "REJECTED"
}
