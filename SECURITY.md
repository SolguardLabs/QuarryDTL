# Security Policy

## Modelo De Seguridad

QuarryDTL modela un motor interno de liquidez donde los vaults exponen
capacidad mediante rutas autorizadas. El sistema separa reservas observadas,
reservas comprometidas, settlements en vuelo, inflows pendientes y previsiones
operativas usadas por el planificador.

El motor asume:

- activos registrados con precision y parametros de riesgo conocidos;
- vaults creados por configuracion interna;
- rutas habilitadas entre vaults compatibles;
- cantidades enteras sin coma flotante;
- lifecycle explicito para reservas, settlements y forecasts;
- salida JSON determinista para auditoria y reproduccion.

## Invariantes Esperadas

- Ningun vault puede tener componentes negativos.
- Ninguna cuenta interna puede tener balances negativos.
- Cada settlement debe referenciar ruta y vaults existentes.
- Cada forecast originado por settlement debe enlazar a un settlement valido.
- Las rutas no pueden liquidar mas de lo asignado.
- Las vistas de capacidad no pueden exponer capacidad negativa.
- Los retiros permanecen abiertos mientras los outflows pendientes esten
  cubiertos por la vista contable del vault.

## Validacion Automatizada

La suite TypeScript valida los escenarios publicos mediante la CLI:

```bash
npm test
```

La validacion completa de CI usa:

```bash
npm run ci
```

## Dependencias

El core Go solo usa la libreria estandar. Node.js se usa para scripts de build
y tests de integracion. Dependabot esta configurado para Go modules, npm y
GitHub Actions.

## Alcance De Revision

El alcance principal es:

- `src/ledger.go`
- `src/planner.go`
- `src/allocator.go`
- `src/rebalance.go`
- `src/liquidation.go`
- `src/risk.go`
- `tests/node/*.test.ts`

## Reportes

Los reportes internos deben incluir:

- escenario afectado;
- salida JSON emitida por la CLI;
- secuencia de rutas, reservas, forecasts y settlements;
- impacto contable o economico;
- propuesta de test de regresion.
