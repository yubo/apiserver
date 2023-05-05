package api

type Demo struct {
	Name string `json:"name" sql:"where"`
	Data string `json:"data"`
}
