# pomocli

Pomodoro de terminal con lista de tareas. Elegís una tarea, corre el timer de foco, suena una alarma al terminar y arranca el descanso. Guarda las tareas y los pomodoros del día en un archivo local.

Está inspirado en [countdown](https://github.com/antonmedv/countdown), de donde sale el estilo de los dígitos grandes.

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
| `-focus` | `25m` | Duración del foco |
| `-short` | `5m` | Duración del descanso corto |
| `-long` | `15m` | Duración del descanso largo |
| `-every` | `4` | Cantidad de focos antes de cada descanso largo |

Las duraciones usan el formato de Go (`90s`, `25m`, `1h`). Tienen que ser mayores que cero y `-every` tiene que ser al menos 1. Si no, el programa sale con código 2.

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

Mientras suena la alarma, cualquier tecla la silencia.

## El ciclo

1. Foco (`-focus`). Al llegar a cero suena la alarma y se guarda el pomodoro.
2. Descanso corto (`-short`).
3. Cada `-every` focos, el descanso es largo (`-long`).
4. Al terminar el descanso suena la alarma y el programa espera que elijas: otro foco, marcar la tarea como hecha o volver a la lista.

## Dónde se guardan los datos

En `%AppData%\pomocli\data.json`. Si el archivo está corrupto, pomocli avisa y sale sin tocarlo.
