package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/text/encoding/charmap"

	"github.com/serg666/service-repository"

	"github.com/serg666/ecom-service/client"
	"github.com/serg666/ecom-service/config"
)

type RateValue float64

func (rv *RateValue) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string

	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}

	v, err := strconv.ParseFloat(strings.Replace(s, ",", ".", -1), 64)
	if err != nil {
		return err
	}

	*rv = RateValue(v)

	return nil
}

type RateDate struct {
	time.Time
}

func (rd *RateDate) UnmarshalXMLAttr(attr xml.Attr) (err error) {
	rd.Time, err = time.Parse("02.01.2006", attr.Value)
	return
}

type Rate struct {
	XMLName     xml.Name  `xml:"Valute"`
	NumericCode int       `xml:"NumCode"`
	CharCode    string    `xml:"CharCode"`
	Nominal     int       `xml:"Nominal"`
	Name        string    `xml:"Name"`
	Value       RateValue `xml:"Value"`
}

type Rates struct {
	XMLName xml.Name `xml:"ValCurs"`
	Date    RateDate `xml:"Date,attr"`
	Rates   []Rate   `xml:"Valute"`
}

func main() {
	cfgPath, err := config.ParseFlags()
	if err != nil {
		log.Fatalf("can not parse flags due to: %v", err)
	}

	cfg, err := config.NewConfig(cfgPath)
	if err != nil {
		log.Fatalf("can not get new config due to: %v", err)
	}

	pgPool, err := repository.MakePgPoolFromDSN(cfg.Databases.Default.Dsn)
	if err != nil {
		log.Fatalf("can not make pg pool: %v", err)
	}

	loggerFunc := func(c interface{}) logrus.FieldLogger {
		return cfg.LogRusLogger(c)
	}

	client.Client = cfg.HttpClient()

	r, err := http.NewRequest("GET", "http://www.cbr.ru/scripts/XML_daily.asp", nil)
	if err != nil {
		log.Fatalf("can not make new request: %v", err)
	}

	res, err := client.Client.Do(r)
	if err != nil {
		log.Fatalf("can not do request: %v", err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("can not read body: %v", err)
	}

	var rates Rates

	reader := bytes.NewReader(body)
	decoder := xml.NewDecoder(reader)

	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch charset {
		case "windows-1251":
			return charmap.Windows1251.NewDecoder().Reader(input), nil
		default:
			return nil, fmt.Errorf("unknown charset: %s", charset)
		}
	}

	if err := decoder.Decode(&rates); err != nil {
		log.Fatalf("can not unmarshal body: %v", err)
	}

	currencyStore := repository.NewPGPoolCurrencyStore(pgPool, loggerFunc)
	rateStore := repository.NewPGPoolRateStore(pgPool, currencyStore, loggerFunc)

	loggerFunc(nil).Printf("start loading currency rates for date: %v", rates.Date)

	rates.Rates = append(rates.Rates, Rate{
		NumericCode: 643,
		CharCode:    "RUB",
		Nominal:     1,
		Name:        "Российский рубль",
		Value:       1,
	})
	rates.Rates = append(rates.Rates, Rate{
		NumericCode: 810,
		CharCode:    "RUR",
		Nominal:     1,
		Name:        "Российский рубль",
		Value:       1,
	})
	for i := 0; i < len(rates.Rates); i++ {
		loggerFunc(nil).Printf("%s (%s): %v", rates.Rates[i].Name, rates.Rates[i].CharCode, rates.Rates[i].Value)
		err, _, currencies := currencyStore.Query(nil, repository.NewCurrencySpecificationByNumericCode(
			rates.Rates[i].NumericCode,
		))

		if err != nil {
			loggerFunc(nil).Warningf("failed to query currency store: %v", err)
			continue
		}

		if len(currencies) == 0 {
			loggerFunc(nil).Warning("currency not found")
			continue
		}

		currency := currencies[0]

		err, _, items := rateStore.Query(nil, repository.NewRateSpecificationByTimeAndCurrency(
			rates.Date.Time,
			currency,
		))

		if err != nil {
			loggerFunc(nil).Warningf("failed to query rate store: %v", err)
			continue
		}

		if len(items) > 0 {
			loggerFunc(nil).Warning("currency rate has already been added")
			continue
		}

		rateValue := float64(rates.Rates[i].Value)
		rate := &repository.Rate{
			Date:     &rates.Date.Time,
			Rate:     &rateValue,
			Nominal:  &rates.Rates[i].Nominal,
			Currency: currency,
		}
		if err := rateStore.Add(nil, rate); err != nil {
			loggerFunc(nil).Warningf("failed to add currency rate: %v", err)
		}
		loggerFunc(nil).Print("currency rate has been added")
	}
}
