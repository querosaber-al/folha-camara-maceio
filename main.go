package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	mainPage = "https://www.camarademaceio.al.gov.br/transparencia/portal/salarios-subsidiosx"
)

var (
	reLinks        = regexp.MustCompile(`http.*ano=\d*`)
	extrairLinks   = flag.Bool("extrair_links", false, "Apenas extrair links para os dados e imprime na saída padrão.")
	ano            = flag.String("ano", "", "Ano de interesse.")
	processarLinks = flag.Bool("processar-links", false, "Apensar processar uma lista de links (um por linha), vindo da entrada padrão.")
)

func main() {
	flag.Parse()

	switch {
	case *extrairLinks:
		// A página da folha de pagamento da câmara não retorna erro se você
		// pedir uma página que não existe. Apenas retorna um doc com zero links.
		for p := 1; ImprimeLinks(fmt.Sprintf("%s&pagina=%d", mainPage, p), *ano) > 0; p++ {
		}
	case *processarLinks:
		writer := csv.NewWriter(os.Stdout)
		cabecalho := []string{
			"matricula", "mes", "ano", "vinculo", "nome", "cargo",
			"lotacao", "remunercao", "abono", "eventuais", "desconto",
			"salario_liquido"}
		writer.Write(cabecalho)
		s := bufio.NewScanner(os.Stdin)
		for s.Scan() {
			ImprimeItemFolha(s.Text(), writer)
		}
		if err := s.Err(); err != nil {
			log.Fatalf("erro lendo entrada padrão:%q", err)
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			log.Fatalf("erro escrevendo na saída padrão:%q", err)
		}
	default:
		fmt.Println("Nada a fazer. Escolha uma das opções:")
		flag.PrintDefaults()
	}
}

// ImprimeItemFolha extrai informações sobre um item na folha de pagamento da
// câmara de vereadores de Maceió.
func ImprimeItemFolha(pagina string, writer *csv.Writer) {
	// Baixando código-fonte da página.
	resp, err := http.Get(pagina)
	if err != nil {
		log.Fatalf("erro baixando página principal:%q", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("código de status inválido:%d(%s)", resp.StatusCode, resp.Status)
	}

	// Extraindo item.
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatalf("erro criando goquery doc:%q", err)
	}
	camposDinheiro := map[string]struct{}{
		"Abono":           struct{}{},
		"Remuneração":     struct{}{},
		"Eventuais":       struct{}{},
		"Desconto":        struct{}{},
		"Salário Líquido": struct{}{},
	}
	var item []string
	doc.Find("tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 { // Tem um primeiro tr para cabeçalho.
			return
		}
		c := s.Find("td").First()
		if c.Text() == "CPF" {
			return
		}
		v := c.Next().Text()

		// Divindo ano e mês de referência em dois campos
		if c.Text() == "Referência" {
			ref := strings.Split(v, " / ")
			item = append(item, ref[0])
			item = append(item, ref[1])
			return
		}

		// Removendo prefixos dos campos que são dinheiro.
		if _, ok := camposDinheiro[c.Text()]; ok {
			d := strings.Split(v, " ")[1]
			d = strings.Replace(d, ".", "", -1)
			d = strings.Replace(d, ",", ".", -1)
			item = append(item, d)
			return
		}
		item = append(item, v)
	})
	writer.Write(item)
}

// ImprimeLinks imprime os links e retorna o número de links impressos.
func ImprimeLinks(pagina, ano string) int {
	// Baixando código-fonte da página.
	resp, err := http.Get(pagina)
	if err != nil {
		log.Fatalf("erro baixando página principal:%q", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("código de status inválido:%d(%s)", resp.StatusCode, resp.Status)
	}

	// Extraindo links.
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatalf("erro criando goquery doc:%q", err)
	}
	n := 0
	doc.Find("tr").Each(func(i int, s *goquery.Selection) {
		if val, ok := s.Attr("onclick"); ok {
			link := reLinks.FindString(val)
			if strings.HasSuffix(link, ano) { // A string vazia é sufixo de todo string.
				n++
				fmt.Println(link)
			}
		}
	})
	return n
}
