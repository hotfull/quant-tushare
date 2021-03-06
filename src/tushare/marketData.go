package tushare

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/jinzhu/gorm"
)

// GetTushareData use http post methord to get data from https://tushare.pro
func (api *TuShare) GetTushareData(apiName string, params Params, fields Fields) (*APIResponse, error) {
	/*
		// Check params
		_, hasTsCode := params["ts_code"]
		_, hasTradeDate := params["trade_date"]

		// ts_code & trade_date required
		if (!hasTsCode && !hasTradeDate) || (hasTsCode && hasTradeDate) {
			return nil, fmt.Errorf("Need one argument ts_code or trade_date")
		}
	*/
	body := map[string]interface{}{
		"api_name": apiName,
		"token":    api.token,
		"fields":   fields,
		"params":   params,
	}
	return api.postData(body)
}

// ParsingTushareData save response of tushare  api  to slice
func ParsingTushareData(resp *APIResponse, dataTypeAddress interface{}, db *gorm.DB) {
	items := resp.Data.Items
	fields := resp.Data.Fields
	for i := 0; i < len(fields); i++ {
		fields[i] = SnakeToUpperCamel(fields[i])
	}

	dbdata := reflect.ValueOf(dataTypeAddress).Elem()
	iterType := reflect.TypeOf(dataTypeAddress).Elem().Elem()
	iterData := reflect.New(iterType)
	for _, value := range items {
		for i := 0; i < len(value); i++ {
			v := reflect.ValueOf(value[i])
			if v.IsValid() {
				iterData.Elem().FieldByName(fields[i]).Set(v)
			}
		}
		dbdata.Set(reflect.Append(dbdata, iterData.Elem()))
	}
}

//UpdateTradeCal function update trade calendar of SSE 、SZSE...
func UpdateTradeCal(db *gorm.DB, api *TuShare) {
	var checkPoint CheckPoint
	var stockExchange StockExchange
	stockExchange = SE
	for _, exchange := range stockExchange {
		checkPoint.Item = exchange
		if err := db.Select("day").Where("item = ?", exchange).Find(&checkPoint).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				checkPoint.Day = "19901219"
			}
		}
		startDate := checkPoint.Day
		endDate := time.Now().Format("20060102")
		params := make(Params)
		fields := APIFullFields["trade_cal"]
		params["exchange"] = exchange
		params["start_date"] = startDate
		params["end_date"] = endDate
		if startDate == endDate {
			fmt.Printf("Trade calendar of %v is already up to date!!!\n", exchange)
		} else {
			if dateFormat := IsDateFormat(params["start_date"], params["end_date"]); !dateFormat {
				log.Fatal("please input right date format YYYYMMDD")
			}
			resp, err := api.GetTushareData("trade_cal", params, fields)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Response of tushare api: ", *resp)
			dataType := []TradeCal{}
			ParsingTushareData(resp, &dataType, db)
			// updata data
			for _, iterData := range dataType {
				if err := db.Find(&iterData).Error; err != nil {
					if err == gorm.ErrRecordNotFound {
						fmt.Printf("Updating %v\n", iterData)
						db.Create(&iterData)
					}
				}
			}
			// update checkPoint
			if err := db.Find(&checkPoint).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					checkPoint.Day = endDate
					db.Create(&checkPoint)
					fmt.Println("Checkpoint create successfully!!!")
				}
			} else {
				db.Delete(&checkPoint)
				checkPoint.Day = endDate
				db.Create(&checkPoint)
				fmt.Println("Checkpoint update successfully!!!")
			}
			fmt.Printf("Trade calendar of %v update successfully!!!\n", exchange)
		}
	}
}

// UpdateDaily update stock daily
func UpdateDaily(db *gorm.DB, api *TuShare) {
	params := make(Params)
	params["start_date"] = "19901219" //default start date
	endDate := time.Now().Format("20060102")
	params["end_date"] = endDate
	fields := APIFullFields["daily"]
	var checkPoints []CheckPoint
	db.Table("check_point").Find(&checkPoints)
	stockList := []StockBasic{}
	if err := db.Table("stock_basic").Find(&stockList).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			fmt.Println("No data found in db!!!")
			fmt.Println("Please update stock basic first!!!")
			return
		}
	}
	for _, stock := range stockList {
		var checkPoint CheckPoint
		params["ts_code"] = stock.TsCode
		flag := 0
		for i := 0; i < len(checkPoints); i++ {
			if stock.TsCode == checkPoints[i].Item {
				checkPoint = checkPoints[i]
				checkPointDay := checkPoints[i].Day
				if checkPointDay == params["end_date"] {
					params["start_date"] = checkPointDay
				} else {
					params["start_date"] = NextDay(checkPointDay)
				}
				checkPoints = append(checkPoints[:i], checkPoints[i+1:]...)
				flag = 1
				break
			}
		}
		if params["start_date"] == params["end_date"] {
			fmt.Printf("Daily data of %v is already up to date!!!\n", stock.TsCode)
			params["start_date"] = "19901219" //reset params["start_date"] to default start date
		} else {
			resp, err := api.GetTushareData("daily", params, fields)
			if err != nil {
				log.Fatal(err)
			}
			params["start_date"] = "19901219" //reset params["start_date"] to default start date
			respData := []Daily{}
			ParsingTushareData(resp, &respData, db)
			fmt.Println("Response data is :", respData) // updata data
			for _, iterData := range respData {
				fmt.Printf("Updating %v\n", iterData)
				db.Create(&iterData)
			}
			// update checkPoint
			if flag == 1 {
				db.Delete(&checkPoint)
				checkPoint.Day = endDate
				db.Create(&checkPoint)
				log.Printf("Checkpoint: %v update successfully!!!", checkPoint)
			} else {
				checkPoint.Day = endDate
				checkPoint.Item = stock.TsCode
				db.Create(&checkPoint)
				log.Printf("Checkpoint: %v create successfully!!!", checkPoint)
			}
		}
	}
}

// UpdateStockBasic update stock list
func UpdateStockBasic(db *gorm.DB, api *TuShare) {
	params := make(Params)
	fields := APIFullFields["stock_basic"]
	resp, err := api.GetTushareData("stock_basic", params, fields)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println("Response of tushare api: ", *resp)
	respData := []StockBasic{}
	ParsingTushareData(resp, &respData, db)
	existData := []StockBasic{}
	if err := db.Find(&existData).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			fmt.Println("No data found in db!!!")
		}
	}
	for _, data := range respData {
		flag := 0
		for i := 0; i < len(existData); i++ {
			if existData[i].TsCode == data.TsCode {
				if existData[i] == data {
					existData = append(existData[:i], existData[i+1:]...)
					flag = 1
					break
				} else {
					db.Delete(existData[i])
				}
			}
		}
		if flag == 1 {
			fmt.Printf("%v already exist!!!\n", data)
		} else {
			fmt.Printf("Updating %v\n", data)
			db.Create(&data)
		}
	}
}
