package ui

import "strings"

const glyphRows = 5

var glyphs = map[rune][glyphRows]string{
	'0': {"█████", "█   █", "█   █", "█   █", "█████"},
	'1': {"  █  ", " ██  ", "  █  ", "  █  ", "█████"},
	'2': {"█████", "    █", "█████", "█    ", "█████"},
	'3': {"█████", "    █", " ████", "    █", "█████"},
	'4': {"█   █", "█   █", "█████", "    █", "    █"},
	'5': {"█████", "█    ", "█████", "    █", "█████"},
	'6': {"█████", "█    ", "█████", "█   █", "█████"},
	'7': {"█████", "    █", "   █ ", "  █  ", "  █  "},
	'8': {"█████", "█   █", "█████", "█   █", "█████"},
	'9': {"█████", "█   █", "█████", "    █", "█████"},
	':': {" ", "█", " ", "█", " "},
}

// BigDigits dibuja s (dígitos y ':') en 5 filas. Los glifos van separados por
// una columna en blanco y todas las filas tienen el mismo ancho.
func BigDigits(s string) string {
	rows := make([]string, glyphRows)
	first := true
	for _, r := range s {
		g, ok := glyphs[r]
		if !ok {
			continue
		}
		for i := range rows {
			if !first {
				rows[i] += " "
			}
			rows[i] += g[i]
		}
		first = false
	}
	return strings.Join(rows, "\n")
}
