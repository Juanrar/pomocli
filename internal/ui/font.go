package ui

import "strings"

// Medidas de la fuente grande (estilo "ANSI Shadow"). Todos los dígitos ocupan
// la misma celda para que la hora no se corra al cambiar un número.
const (
	glyphRows  = 6
	digitCell  = 9
	colonCell  = 3
	glyphGap   = 1
	bigDigitsW = 4*digitCell + colonCell + 4*glyphGap // "MM:SS"
)

var glyphs = map[rune][glyphRows]string{
	'0': {" ██████╗ ", "██╔═████╗", "██║██╔██║", "████╔╝██║", "╚██████╔╝", " ╚═════╝ "},
	'1': {" ██╗", "███║", "╚██║", " ██║", " ██║", " ╚═╝"},
	'2': {"██████╗ ", "╚════██╗", " █████╔╝", "██╔═══╝ ", "███████╗", "╚══════╝"},
	'3': {"██████╗ ", "╚════██╗", " █████╔╝", " ╚═══██╗", "██████╔╝", "╚═════╝ "},
	'4': {"██╗  ██╗", "██║  ██║", "███████║", "╚════██║", "     ██║", "     ╚═╝"},
	'5': {"███████╗", "██╔════╝", "███████╗", "╚════██║", "███████║", "╚══════╝"},
	'6': {" ██████╗ ", "██╔════╝ ", "███████╗ ", "██╔═══██╗", "╚██████╔╝", " ╚═════╝ "},
	'7': {"███████╗", "╚════██║", "    ██╔╝", "   ██╔╝ ", "   ██║  ", "   ╚═╝  "},
	'8': {" █████╗ ", "██╔══██╗", "╚█████╔╝", "██╔══██╗", "╚█████╔╝", " ╚════╝ "},
	'9': {" █████╗ ", "██╔══██╗", "╚██████║", " ╚═══██║", " █████╔╝", " ╚════╝ "},
	':': {"   ", "██╗", "╚═╝", "██╗", "╚═╝", "   "},
}

// center rellena s con espacios hasta w columnas, centrado.
func center(s string, w int) string {
	n := len([]rune(s))
	if n >= w {
		return s
	}
	left := (w - n) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", w-n-left)
}

// BigDigits dibuja s (dígitos y ':') en 6 filas de texto plano. Cada glifo va
// en una celda de ancho fijo, separada de la siguiente por una columna.
func BigDigits(s string) string {
	rows := make([]string, glyphRows)
	first := true
	for _, r := range s {
		g, ok := glyphs[r]
		if !ok {
			continue
		}
		cell := digitCell
		if r == ':' {
			cell = colonCell
		}
		for i := range rows {
			if !first {
				rows[i] += strings.Repeat(" ", glyphGap)
			}
			rows[i] += center(g[i], cell)
		}
		first = false
	}
	return strings.Join(rows, "\n")
}
