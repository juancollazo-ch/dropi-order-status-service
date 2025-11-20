package main

import (
	"fmt"
	"net/http"
)

func main() {
	// Método 1: URL con espacios directamente
	req1, _ := http.NewRequest("GET", "https://api.dropi.co/integrations/orders/myorders?from=2025-11-18&result_number=1&filter_date_by=FECHA DE CAMBIO DE ESTATUS", nil)
	fmt.Println("Método 1 (directo):")
	fmt.Println("  URL.String():", req1.URL.String())
	fmt.Println("  RawQuery:", req1.URL.RawQuery)
	fmt.Println()

	// Método 2: Establecer RawQuery manualmente
	req2, _ := http.NewRequest("GET", "https://api.dropi.co/integrations/orders/myorders", nil)
	req2.URL.RawQuery = "from=2025-11-18&result_number=1&filter_date_by=FECHA DE CAMBIO DE ESTATUS"
	fmt.Println("Método 2 (RawQuery manual):")
	fmt.Println("  URL.String():", req2.URL.String())
	fmt.Println("  RawQuery:", req2.URL.RawQuery)
}
