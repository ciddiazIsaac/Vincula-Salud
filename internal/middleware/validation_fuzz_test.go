package middleware

import (
	"testing"
)

// FuzzValidRunRegex utiliza el motor de Fuzzing de Go para verificar que
// la expresión regular de validación de RUNs no entre en pánico (ReDoS o similares)
// y que maneje correctamente strings de diversas formas.
func FuzzValidRunRegex(f *testing.F) {
	// Casos de prueba iniciales (semillas)
	seeds := []string{
		"12345678-9",
		"1.234.567-K",
		"1234567-k",
		"1234-5",
		"invalido",
		"12.345.678-9-K",
		"",
		"----------",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, run string) {
		// Evaluamos la expresión regular. El objetivo principal del fuzzing aquí 
		// es asegurar que la evaluación no provoque un panic y retorne en un 
		// tiempo razonable independientemente del input (evitar ReDoS).
		_ = validRunRegex.MatchString(run)
	})
}
