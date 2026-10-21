# Contrato JSON de la CLI

`scenario <nombre>` devuelve un objeto JSON con una fotografía completa del ciclo. Los importes se expresan como enteros, los porcentajes como bps y el tiempo como ticks lógicos. Evite convertir cantidades a `float` al consumir la salida.

| Campo | Significado |
| --- | --- |
| `protocol`, `scenario`, `network_id`, `clock` | Identidad y reloj lógico de la ejecución. |
| `state_digest` | Huella determinista del ledger. |
| `vaults`, `routes`, `settlements` | Entidades y estados actuales. |
| `forecasts`, `views` | Previsiones y vistas del planificador. |
| `exposure` | Agregado por ruta de demanda, allocation, submitted, finalized, failed y fees. |
| `totals` | Buckets por activo de reservas, pasivos, entradas, salidas y fees. |
| `risk`, `invariants` | Métricas e indicadores de validación. |
| `events`, `notes` | Secuencia operacional y anotaciones del escenario. |

La clave `exposure.utilization_bps` usa demanda como denominador. Una ruta sin demanda devuelve cero. `fees_finalized` excluye los settlements que aún no terminaron. Las integraciones deben asociar cada agregado al `state_digest` y no sumar activos con distintas unidades.

Ejemplo de inspección local:

```bash
out/quarrydtl scenario allocation | jq '{scenario, state_digest, exposure, invariants}'
```

`jq` es opcional; la CLI no lo requiere. Si no está instalado, guarde la salida JSON y procese el documento con cualquier parser compatible.
