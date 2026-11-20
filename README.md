# pomocli

Pomodoro de terminal con lista de tareas. Elegís una tarea, corre el timer de foco, suena una alarma al terminar y arranca el descanso. Guarda las tareas y todo el historial de pomodoros en un archivo local.

Está inspirado en [countdown](https://github.com/antonmedv/countdown), de donde sale la idea de los dígitos grandes. Los dígitos usan el estilo "ANSI Shadow": cuerpo gris claro con sombra gris más oscura. El color de la línea de fase distingue el foco (rojo) del descanso (verde).

## Instalación

Necesitás Go instalado.

```
go install github.com/Juanrar/pomocli@latest
```

Esto deja `pomocli.exe` en `%USERPROFILE%\go\bin`. Agregá esa carpeta al `PATH` si todavía no está.

## Uso

```
pomocli [flags]
```

| Flag | Por defecto | Qué hace |
|------|-------------|----------|
| `-focus` | `50m` | Duración del foco |
| `-short` | `10m` | Duración del descanso corto |
| `-long` | `10m` | Duración del descanso largo |
| `-every` | `4` | Cantidad de focos antes de cada descanso largo |

Los valores también se editan desde la pantalla de configuración (`c` en la lista). Se guardan en el mismo archivo de datos y se usan en las próximas sesiones. Un flag pasado en la línea de comandos gana sobre lo guardado, pero solo para esa ejecución y sin cambiar lo guardado.

`-h` muestra la ayuda. pomocli no acepta argumentos sueltos, solo flags. Las duraciones usan el formato de Go (`90s`, `25m`, `1h`). Tienen que ser mayores que cero y `-every` tiene que ser al menos 1. Si no, el programa sale con código 2.

Para probar el ciclo completo rápido:

```
pomocli -focus 5s -short 3s -long 4s -every 2
```

## Teclas

### Lista de tareas

| Tecla | Acción |
|-------|--------|
| `↑` / `k` | Subir |
| `↓` / `j` | Bajar |
| `a` | Agregar tarea |
| `c` | Configurar duraciones (`↑↓` elegir, `←→` cambiar, `esc` volver) |
| `e` | Editar el título |
| `enter` | Empezar un foco sobre la tarea |
| `+` / `-` | Subir o bajar la estimación de pomodoros |
| `d` | Marcar o desmarcar como hecha |
| `x` | Borrar (pide confirmación con `s` o `n`) |
| `r` | Resumen de hoy |
| `q` / `ctrl+c` | Salir |

Al agregar o editar, `enter` confirma y `esc` cancela.

### Timer de foco

| Tecla | Acción |
|-------|--------|
| `espacio` | Pausar o reanudar |
| `esc` | Cancelar el foco y volver a la lista |
| `q` / `ctrl+c` | Salir (pide confirmación con `s` o `n`) |

### Descanso

| Tecla | Acción |
|-------|--------|
| `espacio` | Pausar o reanudar |
| `s` | Saltar el descanso |
| `esc` | Volver a la lista |
| `q` / `ctrl+c` | Salir |

### Después del descanso

| Tecla | Acción |
|-------|--------|
| `enter` | Otro foco sobre la misma tarea |
| `d` | Marcar la tarea como hecha y volver a la lista |
| `esc` | Volver a la lista |
| `q` / `ctrl+c` | Salir |

### Resumen de hoy

| Tecla | Acción |
|-------|--------|
| `esc` | Volver a la lista |
| `ctrl+c` | Salir |

Mientras suena la alarma, cualquier tecla la silencia. La alarma se corta sola a los 60 segundos.

Si no hay dispositivo de audio al abrir pomocli, usa el bell de la terminal y muestra un aviso. Conectá los auriculares antes de abrirlo.

## El ciclo

1. Foco (`-focus`). Al llegar a cero suena la alarma y se guarda el pomodoro.
2. Descanso corto (`-short`).
3. Cada `-every` focos, el descanso es largo (`-long`).
4. Al terminar el descanso suena la alarma y el programa espera que elijas: otro foco, marcar la tarea como hecha o volver a la lista.

## Dónde se guardan los datos

En `%AppData%\pomocli\data.json`. El archivo guarda todo el historial. Si el archivo está corrupto, pomocli avisa y sale sin tocarlo.
