package api

type Demo struct {
	Id   *int   `json:"id" sql:",primary_key,auto_increment=1000"`
	Name string `json:"name" sql:",where"`
	Data string `json:"data"`
}
