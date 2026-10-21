# Modelo económico

Los importes del motor son enteros en la unidad interna de cada activo. No se usa coma flotante para saldos. Las tarifas y umbrales se expresan en puntos básicos; `10 000 bps = 100 %`. Para convertir a unidades visuales, consulte los decimales del activo, sin redondear el saldo de ledger.

## Distribución de capacidad

Sea `R` la reserva del vault, `I` su idle, `b` el buffer mínimo, `F` la previsión admitida, `D` la demanda pendiente, `q` la cuota máxima de ruta y `m` el lote mínimo:

```text
b = floor(R × min_vault_buffer_bps / 10 000)
libre = max(I − b, 0)
asignable = libre + F
capacidad_ruta = min(asignable, floor(base × max_route_share_bps / 10 000), q, D)
plan = capacidad_ruta si capacidad_ruta ≥ m; en otro caso 0
```

`base` es la reserva observada y, cuando la política admite previsiones, incorpora la previsión visible. Este límite por ruta evita que una ruta monopolice toda la vista de un vault; no es un límite global entre todas las rutas. La previsión es un dato de planificación y no un depósito finalizado.

Ejemplo: un vault con `R = 1 000 000`, `I = 900 000`, buffer del 7 %, previsión nula, cuota de ruta del 65 % y demanda `800 000` tiene `libre = 830 000`; el límite de ruta es `650 000`. Si el lote mínimo es `100 000`, el plan es `650 000`.

## Settlement y fees

Una salida de importe bruto `A` registra un outflow de `A`. El destino espera el neto `A − fee`. La tarifa se reconoce en el informe únicamente cuando el settlement finaliza. `BuildRouteExposures` suma por ruta importes submitted, finalized y failed; `utilization_bps = allocated × 10 000 / demand`. La exposición presentada es una medida de libro, no una valoración de mercado.

## Cobertura y liquidación

```text
cobertura_bps = floor(reserva × 10 000 / pasivo)
reserva_requerida = floor(pasivo × liquidation_threshold_bps / 10 000)
shortfall = max(reserva_requerida − reserva, 0)
penalización = floor(shortfall × recovery_penalty_bps / 10 000)
```

El motor recupera como máximo el idle disponible y separa penalización de pasivo amortizado. Por ejemplo, con pasivo `900 000`, umbral de `9 250 bps` y reserva `650 000`, la reserva requerida es `832 500` y el shortfall `182 500`; al `250 bps` de penalización, el valor teórico es `4 562` en unidades internas, sujeto al idle recuperable.

## Límites del modelo

El riesgo de precio, slippage, fees de red, latencia variable de proveedores y conciliación externa no se estiman en este motor. Cualquier integración que use estas cifras debe definir controles propios antes de interpretar un resultado local como saldo liquidado.
