package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pterm/pterm"
)

type (
	Position struct {
		Ticket string  `json:"ticket"`
		Amount float64 `json:"amount"`
	}

	Operation struct {
		Op     string
		Ticket string
		Amount float64
	}
)

func tickets(portfolio []Position, ranking []string) []string {
	tickets := make([]string, 0)
	tickets = append(tickets, ranking...)

	for _, p := range portfolio {
		tickets = append(tickets, p.Ticket)
	}

	return tickets
}

func getPrice(prices map[string]float64, ticket string) float64 {
	price, found := prices[ticket]
	if !found {
		log.Printf("%s does not have a price\n", ticket)

		return 0.0
	}

	return price
}

func getPortfolioValue(portfolio []Position, prices map[string]float64) float64 {
	sum := 0.0
	for _, p := range portfolio {
		sum += (getPrice(prices, p.Ticket) * p.Amount)
	}

	return sum
}

func getAmount(portfolio []Position, ticket string) float64 {
	for _, p := range portfolio {
		if p.Ticket == ticket {
			return p.Amount
		}
	}

	return 0.0
}

func fetchCurrentPrices(tickets []string) map[string]float64 {
	/**
	 * Source: https://brapi.dev/docs
	 */
	type response struct {
		Results []struct {
			Symbol             string
			RegularMarketPrice float64
		}
	}

	resp, err := http.Get(fmt.Sprintf("https://brapi.dev/api/quote/%s", strings.Join(tickets, ",")))
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	var parsed response

	err = json.NewDecoder(resp.Body).Decode(&parsed)
	if err != nil {
		panic(err)
	}

	output := make(map[string]float64)
	for _, p := range parsed.Results {
		output[p.Symbol] = p.RegularMarketPrice
	}

	return output
}

func sortTable(data [][]string) func(int, int) bool {
	return func(i, j int) bool {
		current, _ := strconv.ParseFloat(data[i][3], 64)
		next, _ := strconv.ParseFloat(data[j][3], 64)

		return current > next
	}
}

func renderTable(data [][]string, title string) string {
	table := [][]string{
		{"Ticket", "Quantidade", "Preço", "Total"},
	}
	table = append(table, data...)

	rendered, err := pterm.
		DefaultTable.
		WithHasHeader().
		WithData(table).
		Srender()

	if err != nil {
		return "Não foi possível renderizar a tabela " + strings.ToLower(title)
	}

	return pterm.DefaultBox.WithTitle(title).WithTitleTopCenter().Sprint(rendered)
}

func renderPortfolio(title string, portfolio []Position, prices map[string]float64) string {
	data := make([][]string, 0)

	for _, p := range portfolio {
		price := getPrice(prices, p.Ticket)

		data = append(data, []string{
			p.Ticket,
			strconv.FormatInt(int64(p.Amount), 10),
			fmt.Sprintf("%.2f", price),
			fmt.Sprintf("%.2f", price*p.Amount),
		})
	}

	sort.Slice(data, sortTable(data))

	return renderTable(data, title)
}

func renderOperations(operations []Operation) string {
	table := make([][]string, 1+len(operations))
	table[0] = []string{
		"Operação",
		"Ticket",
		"Quantidade",
	}

	for i, op := range operations {
		title := pterm.LightRed("Vender")
		if op.Op == "buy" {
			title = pterm.LightGreen("Comprar")
		}

		table[i+1] = []string{
			title,
			op.Ticket,
			fmt.Sprintf("%d", int(op.Amount)),
		}
	}

	rendered, err := pterm.
		DefaultTable.
		WithHasHeader().
		WithData(table).
		Srender()

	if err != nil {
		log.Println(err.Error())

		return "Ocorreu um erro ao renderizar as operações"
	}

	return pterm.DefaultBox.WithTitle("Operações").WithTitleTopLeft().Sprint(rendered)
}

func render(original, balanced []Position, operations []Operation, prices map[string]float64, money float64) {
	pterm.DefaultBasicText.Println()
	pterm.DefaultBasicText.Println("Valor da alocação: " + pterm.LightGreen(fmt.Sprintf("%.2f", money)))

	panels := pterm.Panels{
		{
			pterm.Panel{Data: renderPortfolio("Carteira atual", original, prices)},
			pterm.Panel{Data: renderPortfolio("Carteira rebalanceada", balanced, prices)},
		},
		{},
		{
			pterm.Panel{Data: renderOperations(operations)},
		},
	}

	_ = pterm.DefaultPanel.WithPanels(panels).WithPadding(5).Render()
}

func distribute(portfolio []Position, ranking []string, money float64, prices map[string]float64) ([]Position, []Operation) {
	balanced := make([]Position, 0)
	operations := make([]Operation, 0)

	finalBalance := money + getPortfolioValue(portfolio, prices)
	target := finalBalance / float64(len(ranking))

	visited := make(map[string]bool)

	for _, ticket := range ranking {
		if finalBalance == 0 {
			break
		}

		visited[ticket] = true

		ticketPrice := getPrice(prices, ticket)

		currentAmount := getAmount(portfolio, ticket)

		currentValue := ticketPrice * currentAmount

		required := math.Abs(target - currentValue)

		if required > finalBalance {
			required = finalBalance
		}

		operationAmount := math.Floor(required / ticketPrice)

		if operationAmount > 100 {
			operationAmount -= math.Mod(operationAmount, 100)
		}

		operationValue := math.Floor(operationAmount * ticketPrice)

		position := Position{
			Ticket: ticket,
			Amount: currentAmount,
		}

		if currentValue >= target && operationAmount >= 100 { // too much, let's sell some positions
			position.Amount = currentAmount - operationAmount

			balanced = append(balanced, position)

			operations = append(operations, Operation{
				Op:     "sell",
				Ticket: ticket,
				Amount: operationAmount,
			})

			finalBalance += operationValue

			continue
		}

		if target >= currentValue {
			position.Amount = currentAmount + operationAmount

			balanced = append(balanced, position)

			operations = append(operations, Operation{
				Op:     "buy",
				Ticket: ticket,
				Amount: operationAmount,
			})

			finalBalance -= operationValue

			continue
		}
	}

	for _, position := range portfolio {
		if _, ok := visited[position.Ticket]; ok {
			continue
		}

		operations = append(operations, Operation{
			Op:     "sell",
			Ticket: position.Ticket,
			Amount: position.Amount,
		})
	}

	return balanced, operations
}

func parseAndDistribute(data io.Reader, ranking []string, money float64) ([]Position, []Position, []Operation, map[string]float64) {
	portfolio := make([]Position, 0)

	err := json.NewDecoder(data).Decode(&portfolio)
	if err != nil {
		return nil, nil, nil, nil
	}

	prices := fetchCurrentPrices(tickets(portfolio, ranking))

	balanced, operations := distribute(portfolio, ranking, money, prices)

	return portfolio, balanced, operations, prices
}

func getPreviousFileName() string {
	now := time.Now()
	year := now.Year()
	currentMonth := now.Month()
	month := currentMonth - 1

	if currentMonth == time.January {
		month = time.December
		year--
	}

	return strings.ToLower(fmt.Sprintf("%d-%02s.json", year, month))
}

func getCurrentFileName() string {
	now := time.Now()
	year := now.Year()
	month := now.Month()

	return strings.ToLower(fmt.Sprintf("%d-%02s.json", year, month))
}

func save(filename string, balanced []Position) {
	encoded := bytes.NewBuffer(make([]byte, 0))

	encoder := json.NewEncoder(encoded)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(balanced); err != nil {
		panic(err)
	}

	if err := os.WriteFile(filename, encoded.Bytes(), 0644); err != nil {
		panic(err)
	}
}

func getRanking() []string {
	data, err := os.Open("ranking.json")

	if err != nil {
		log.Fatal(err)
	}
	defer data.Close()

	ranking := make([]string, 0)

	_ = json.NewDecoder(data).Decode(&ranking)

	return ranking
}

func main() {
	var file string

	flag.StringVar(&file, "file", getPreviousFileName(), "the portfolio file")
	flag.Parse()

	if len(os.Args) == 1 {
		fmt.Println("Usage: distribute <amount>")
		os.Exit(1)
	}

	money, err := strconv.ParseFloat(os.Args[1], 64)
	if err != nil {
		panic(err)
	}

	data, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer data.Close()

	original, balanced, operations, prices := parseAndDistribute(data, getRanking(), money)

	render(original, balanced, operations, prices, money)
	save(getCurrentFileName(), balanced)
}
