package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/MontFerret/ferret/pkg/compiler"
	"github.com/MontFerret/ferret/pkg/drivers"
	"github.com/MontFerret/ferret/pkg/drivers/cdp"
	fhttp "github.com/MontFerret/ferret/pkg/drivers/http"
)

type Giornata struct {
	Giornata    string      `json:"giornata"`
	URL         string      `json:"url"`
	MatchReport MatchReport `json:"match_report"`
}

type MatchReport struct {
	Giornata string `json:"giornata"`
	PDF      string `json:"pdf"`
}

func main() {
	base := "http://www.legaseriea.it"
	// andata only gets 1st half of the season
	andata := `
		LET base = "http://www.legaseriea.it"
		LET ris = DOCUMENT(base + "/it/serie-a/calendario-e-risultati")
		FOR el1 IN ELEMENTS(ris, ".box_Ngiornata_andata")
			LET giornata = ELEMENT(el1, "a")
			LET url1 = base + giornata.attributes.href
			LET mr = DOCUMENT(base + giornata.attributes.href)
			FOR el2 IN ELEMENTS(mr, ".link-matchreport")
				LET url2 = ELEMENT(el2, "a")
		RETURN {
			giornata: TRIM(giornata.innerText),
			url: url1,
			match_report: {
				giornata: TRIM(giornata.innerText),
				pdf: url2.attributes.href
			}
		}
	`
	c := compiler.New()
	p, err := c.Compile(andata)
	if err != nil {
		log.Println(err)
	}

	ctx := context.Background()
	ctx = drivers.WithContext(ctx, cdp.NewDriver())
	ctx = drivers.WithContext(ctx, fhttp.NewDriver(), drivers.AsDefault())

	out, err := p.Run(ctx)
	if err != nil {
		log.Println(err)
	}

	gs := make([]*Giornata, 0, 19)
	err = json.Unmarshal(out, &gs)
	if err != nil {
		log.Println(err)
	}

	for _, g := range gs {
		pdf := strings.Replace(g.MatchReport.PDF, "program", "report", 1)

		res, err := http.Get(base + pdf)
		if err != nil {
			log.Printf("failed getting resource from %s%s", base, pdf)
		}
		defer res.Body.Close()

		name := strings.Split(g.MatchReport.PDF, "/")
		f, err := os.Create(fmt.Sprintf("%s-%s-%s.pdf", name[4], name[7], name[8]))
		if err != nil {
			log.Printf("failed creating resource %s-%s-%s.pdf", name[4], name[7], name[8])
		}

		_, err = io.Copy(f, res.Body)
		if err != nil {
			log.Printf("failed copying to resource %s-%s-%s.pdf", name[4], name[7], name[8])
		}
	}
}
