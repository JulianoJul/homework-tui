# Homework — versión Go

Organizador de tareas y materiales universitarios para la terminal, con el tema Gruvbox definido en `../DESIGN.md`. Detecta como materias las carpetas inmediatas dentro de `~/Documentos/homework`; no modifica los materiales.

El panel de materiales organiza cada materia por carpeta numerada, sin interpretar esos números como progreso. Con el foco en materiales, `,` / `.` (o `[` / `]`) cambia de actividad y `←` / `→` cambia entre **Mis archivos** (`created_by_me`), **Material de apoyo** (`teacher_resources`) y, si contiene archivos, **Otros**. `General` contiene los archivos que están fuera de las carpetas numeradas. Las flechas arriba/abajo seleccionan un archivo y `Enter` lo abre. Las subcarpetas se recorren de forma recursiva.

## Ejecutar

```sh
cd ~/Documentos/projects/tui/go
go run .
```

Se recomienda una terminal de al menos 80 × 20 y con TrueColor. Para compilar un ejecutable local:

```sh
go build -o homework .
./homework
```

Para usar otra carpeta de materias: `go run . -root /ruta/a/materias`. La aplicación detecta nuevas carpetas al arrancar o al pulsar `r`; no elimina automáticamente materias si sus carpetas desaparecen.

## Controles

- `Tab` / `Shift+Tab`: cambiar entre materias, tareas y materiales; flechas, `Inicio`, `Fin`, `Re Pág`, `Av Pág`: navegar.
- `1` / `2`: seleccionar el panel de tareas / materiales. Ambos están visibles. `Enter`: editar tarea o abrir archivo.
- `Ctrl+N` o `n`: crear tarea. En el formulario, `Tab` o las flechas arriba/abajo cambian de campo; en «Materia», «Prioridad» y «Estado», las flechas izquierda/derecha o espacio eligen el valor. `Ctrl+S` o `Enter` guarda y `Esc` cancela. También se puede cambiar la materia y completar una tarea al editar.
- `Espacio`: completar o reabrir tarea; `c`: mostrar/ocultar completadas; `d` o `Supr`: eliminar con confirmación.
- Al editar una tarea, `Ctrl+D` abre la confirmación para eliminarla; al cancelar se conserva el formulario.
- `g`: alternar entre tareas de todas las materias y tareas de la materia seleccionada. Los materiales siguen mostrando la materia seleccionada en el panel izquierdo.
- Las tareas pendientes con fecha muestran hasta siete círculos, calculados desde el día de creación hasta la entrega; se van vaciando con el tiempo. Al quedar un día o menos aparece un círculo relleno y un aviso rojo. No representan el avance real del trabajo. Las tareas sin fecha o completadas no muestran círculos.
- `Ctrl+F` o `/`: filtrar la vista actual. `Enter` aplica; `Esc` limpia.
- `r`, `Ctrl+R` o `F5`: releer materias y materiales; `O` mayúscula: abrir la carpeta de la materia. La aplicación también comprueba la raíz y los archivos de la materia seleccionada cada 3 segundos. Las materias que desaparecen no se borran automáticamente, para conservar sus tareas.
- `?` o `F1`: ayuda; `Ctrl+Q` o `q`: salir.

El ratón permite seleccionar paneles con un clic; al pasar sobre uno, su borde se resalta en aqua, distinto al naranja del panel activo. La selección de filas, formularios y acciones se hace con teclado. Para seleccionar texto en el terminal, normalmente se puede mantener `Shift` (depende del emulador). La captura de movimiento del ratón depende del soporte del terminal. Los archivos se abren con `xdg-open` en Linux, usando la aplicación predeterminada del escritorio.

## Datos y configuración

Las tareas se guardan en `~/.local/share/homework/data.json` (o `$XDG_DATA_HOME/homework/data.json`). El guardado reemplaza el archivo de forma atómica. Haz copias de seguridad de ese archivo; no ejecutes dos instancias a la vez porque no hay bloqueo entre procesos.

Opcionalmente, crea `~/.config/homework/config.json` (o `$XDG_CONFIG_HOME/homework/config.json`):

```json
{
  "root": "~/Documentos/homework",
  "ignore": ["go", "python", ".git"],
  "openers": {"pdf": ["okular"]}
}
```

Los comandos de `openers` son programas gráficos: la ruta se añade como último argumento, sin pasar por un shell. `HOMEWORK_ROOT`, `HOMEWORK_CONFIG` y `HOMEWORK_DATA` permiten cambiar las rutas; `-root` y `-data` tienen prioridad para esa ejecución. Si existe un archivo de configuración, su `root` tiene prioridad sobre `HOMEWORK_ROOT`.

Esta es solo la primera versión solicitada, en Go. La versión Python queda pendiente.
