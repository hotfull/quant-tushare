package tushare

import "fmt"

// GetDaily 获取股票行情数据, 日线行情
func (api *TuShare) GetDaily(params map[string]string, fields []string) (*APIResponse, error) {
	// Check params
	_, hasTsCode := params["ts_code"]
	_, hasTradeDate := params["trade_date"]

	// ts_code & trade_date required
	if (!hasTsCode && !hasTradeDate) || (hasTsCode && hasTradeDate) {
		return nil, fmt.Errorf("Need one argument ts_code or trade_date")
	}

	if dateFormat := IsDateFormat(params["trade_date"], params["start_date"], params["end_date"]); !dateFormat {
		return nil, fmt.Errorf("please input right date format YYYYMMDD")
	}

	body := map[string]interface{}{
		"api_name": "daily",
		"token":    api.token,
		"fields":   fields,
		"params":   params,
	}

	return api.postData(body)
}

// GetTradeCal get trade calendar of SSE or SZSE
func (api *TuShare) GetTradeCal(params map[string]string, fields []string) (*APIResponse, error) {
	// Check params

	if dateFormat := IsDateFormat(params["start_date"], params["end_date"]); !dateFormat {
		return nil, fmt.Errorf("please input right date format YYYYMMDD")
	}

	body := map[string]interface{}{
		"api_name": "trade_cal",
		"token":    api.token,
		"fields":   fields,
		"params":   params,
	}

	return api.postData(body)
}
