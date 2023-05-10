# distributer

That's my little program to help me rebalance my stock portfolio.

## How to run it

1. Create a `ranking.json` with your ranking, like this example:

```json
["PETR4", "MOVI3", "VALE3", "QUAL3"]
```

This file will change once per year after your first investment, as the defined rule by Clube do Valor.

2. Create a file with your current portfolio, the filename must be in the format `YYYY-month.json`, example:

Filename assuming it's May 2023: `2023-april.json`

```json
[
  {"ticket": "QUAL3", "amount": 1000},
  {"ticket": "DXCO3", "amount": 100}
]
```

3. Run the command: `go run ./distributer <amount>`, replacing `<amount>` by the value that you want to invest.

It will display the current portfolio  and the balanced portfolio, it will also save a file to the next month (`2023-may.json`), so you don't need to repeat the step 2.

### Example

Output of running it with the files of this example and investing more 5000.00:

```
Valor da alocação: 5000.00


┌─────────── Carteira atual ────────────┐     ┌──────── Carteira rebalanceada ────────┐
| Ticket | Quantidade | Preço | Total   |     | Ticket | Quantidade | Preço | Total   |
| QUAL3  | 1000       | 4.34  | 4340.00 |     | QUAL3  | 600        | 4.34  | 2604.00 |
| DXCO3  | 100        | 7.15  | 715.00  |     | VALE3  | 35         | 70.69 | 2474.15 |
|                                       |     | PETR4  | 100        | 24.59 | 2459.00 |
└───────────────────────────────────────┘     | MOVI3  | 200        | 9.77  | 1954.00 |
                                              |                                       |
                                              └───────────────────────────────────────┘
┌─ Operações ──────────────────┐
| Operação | Ticket | Quantidade |
| Comprar  | PETR4  | 100        |
| Comprar  | MOVI3  | 200        |
| Comprar  | VALE3  | 35         |
| Vender   | QUAL3  | 400        |
| Vender   | DXCO3  | 100        |
|                                |
└────────────────────────────────┘
```
