# Arquitectura y ciclo de estados

El ledger mantiene siete dominios: registro de activos, cuentas, vaults, rutas, settlements, reservas, previsiones y diario. Cada ejecución empieza desde una semilla reproducible; el reloj lógico avanza por ticks y no depende del reloj de la máquina.

```mermaid
sequenceDiagram
    participant O as Operación
    participant P as Planificador
    participant L as Ledger
    participant S as Settlement
    participant R as Riesgo
    O->>P: Solicitar ciclo y modo de capacidad
    P->>L: Consultar vaults y rutas abiertas
    L-->>P: Reservas, demanda y previsiones
    P-->>O: Plan ordenado por puntuación
    O->>L: Ejecutar batch
    L->>S: Preparar y enviar settlement
    S->>L: Finalización o fallo
    L->>R: Diario, exposición e invariantes
    R-->>O: Informe JSON y digest
```

`PlannerOptions` decide si la vista usa solo liquidez observada o también previsiones admitidas. `RouteCapacity` cruza la capacidad del vault con la cuota máxima de ruta y su demanda pendiente. `Allocator` produce planes y ejecuta cada uno contra `Ledger.OpenSettlement`. Un intent de rebalanceo limita a ocho settlements y tiene un lote mínimo propio.

```mermaid
stateDiagram-v2
    [*] --> open
    open --> prepared
    prepared --> submitted
    submitted --> finalized
    submitted --> failed
    submitted --> cancelled
    finalized --> [*]
    failed --> [*]
    cancelled --> [*]
```

El estado terminal determina cómo se computan `exposure.finalized`, `exposure.failed` y `fees_finalized`. Una previsión tiene ciclo independiente (`scheduled`, `observed`, `settled`, `voided`), identificado por `source_settlement`.

## Reproducibilidad

`state_digest` resume el estado serializado del ledger. Dos ejecuciones del mismo escenario, versión y configuración deben producir el mismo digest. Los IDs del secuenciador y el ordenamiento de rutas también son deterministas; una diferencia inesperada es motivo para revisar el commit, el escenario y la configuración antes de comparar importes.
