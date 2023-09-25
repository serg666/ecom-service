package main

import (
	"encoding/xml"
	"flag"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/serg666/service-repository"

	"github.com/serg666/ecom-service/config"
)

type Currency struct {
	XMLName     xml.Name `xml:"ISO_CURRENCY"`
	Entity      string   `xml:"ENTITY"`
	Name        string   `xml:"CURRENCY"`
	CharCode    string   `xml:"ALPHABETIC_CODE"`
	NumericCode int      `xml:"NUMERIC_CODE"`
	Exponent    string   `xml:"MINOR_UNIT"`
}

type Currencies struct {
	XMLName    xml.Name   `xml:"ISO_CCY_CODES"`
	Currencies []Currency `xml:"ISO_CURRENCY"`
}

func main() {
	cfgPath := flag.String("config", "./config.yml", "path to config file")
	currenciesPath := flag.String("currencies", "./currencies.xml", "path to currencies file")
	flag.Parse()

	if err := config.ValidateConfigPath(cfgPath); err != nil {
		log.Fatalf("can not validate config path: %v", err)
	}

	if err := config.ValidateConfigPath(currenciesPath); err != nil {
		log.Fatalf("can not validate currencies path: %v", err)
	}

	cfg, err := config.NewConfig(cfgPath)
	if err != nil {
		log.Fatalf("can not get new config due to: %v", err)
	}

	pgPool, err := repository.MakePgPoolFromDSN(cfg.Databases.Default.Dsn)
	if err != nil {
		log.Fatalf("Can not make pg pool: %v", err)
	}

	loggerFunc := func(c interface{}) logrus.FieldLogger {
		return cfg.LogRusLogger(c)
	}

	currencyStore := repository.NewPGPoolCurrencyStore(pgPool, loggerFunc)

	xmlFile, err := os.Open(*currenciesPath)
	if err != nil {
		log.Fatalf("can not open currencies file: %v", err)
	}
	defer xmlFile.Close()

	byteValue, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		log.Fatalf("can not read xml file: %v", err)
	}

	var currencies Currencies
	xml.Unmarshal(byteValue, &currencies)

	loggerFunc(nil).Print("Start loading")
	for i := 0; i < len(currencies.Currencies); i++ {
		charCode := currencies.Currencies[i].CharCode
		numericCode := currencies.Currencies[i].NumericCode
		if len(charCode) > 0 && numericCode > 0 {
			var exp *int
			exponent, err := strconv.Atoi(currencies.Currencies[i].Exponent)

			if err == nil && exponent > 0 {
				res := int(math.Pow10(exponent))
				exp = &res
			}

			err, _, items := currencyStore.Query(nil, repository.NewCurrencySpecificationByNumericCode(numericCode))
			if err == nil {
				if len(items) == 0 {
					currency := &repository.Currency{
						NumericCode: &numericCode,
						Name:        &currencies.Currencies[i].Name,
						CharCode:    &charCode,
						Exponent:    exp,
					}
					if err := currencyStore.Add(nil, currency); err != nil {
						loggerFunc(nil).Printf("Error adding currency by code (%d): %v", numericCode, err)
					} else {
						loggerFunc(nil).Printf("Added currency: %d (%s)", numericCode, charCode)
					}
				}
			} else {
				loggerFunc(nil).Printf("Error quering currency by code (%d): %v", numericCode, err)
			}
		}
	}
	loggerFunc(nil).Print("Stop loading")
}
