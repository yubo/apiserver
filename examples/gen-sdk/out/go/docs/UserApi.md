# \UserApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**Func1**](UserApi.md#Func1) | **Post** /api/user | create user
[**Func2**](UserApi.md#Func2) | **Get** /api/user | search/list users
[**Func3**](UserApi.md#Func3) | **Get** /api/user/{user-name} | get user
[**Func4**](UserApi.md#Func4) | **Put** /api/user/{user-name} | update user
[**Func5**](UserApi.md#Func5) | **Delete** /api/user/{user-name} | delete user



## Func1

> MainCreateUserOutput Func1(ctx).Body(body).Execute()

create user

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "./openapi"
)

func main() {
    body := *openapiclient.NewMainCreateUserInput("Name_example", "NickName_example", "Phone_example") // MainCreateUserInput | 

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.UserApi.Func1(context.Background()).Body(body).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `UserApi.Func1``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `Func1`: MainCreateUserOutput
    fmt.Fprintf(os.Stdout, "Response from `UserApi.Func1`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiFunc1Request struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**MainCreateUserInput**](MainCreateUserInput.md) |  | 

### Return type

[**MainCreateUserOutput**](MainCreateUserOutput.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## Func2

> MainGetUsersOutput Func2(ctx).Query(query).Count(count).PageSize(pageSize).CurrentPage(currentPage).Sorter(sorter).Order(order).Execute()

search/list users

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "./openapi"
)

func main() {
    query := "query_example" // string | query user (optional)
    count := true // bool | just response total count (optional)
    pageSize := int32(56) // int32 | page size (optional)
    currentPage := int32(56) // int32 | current page number, start at 1(defualt) (optional)
    sorter := "sorter_example" // string | column name (optional)
    order := "order_example" // string | asc(default)/desc (optional)

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.UserApi.Func2(context.Background()).Query(query).Count(count).PageSize(pageSize).CurrentPage(currentPage).Sorter(sorter).Order(order).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `UserApi.Func2``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `Func2`: MainGetUsersOutput
    fmt.Fprintf(os.Stdout, "Response from `UserApi.Func2`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiFunc2Request struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **query** | **string** | query user | 
 **count** | **bool** | just response total count | 
 **pageSize** | **int32** | page size | 
 **currentPage** | **int32** | current page number, start at 1(defualt) | 
 **sorter** | **string** | column name | 
 **order** | **string** | asc(default)/desc | 

### Return type

[**MainGetUsersOutput**](MainGetUsersOutput.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## Func3

> MainUser Func3(ctx, userName).Execute()

get user

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "./openapi"
)

func main() {
    userName := "userName_example" // string | 

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.UserApi.Func3(context.Background(), userName).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `UserApi.Func3``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `Func3`: MainUser
    fmt.Fprintf(os.Stdout, "Response from `UserApi.Func3`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**userName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFunc3Request struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**MainUser**](MainUser.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## Func4

> MainUser Func4(ctx, userName).Body(body).Execute()

update user

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "./openapi"
)

func main() {
    userName := "userName_example" // string | 
    body := *openapiclient.NewMainUpdateUserBody("NickName_example", "Phone_example") // MainUpdateUserBody | 

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.UserApi.Func4(context.Background(), userName).Body(body).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `UserApi.Func4``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `Func4`: MainUser
    fmt.Fprintf(os.Stdout, "Response from `UserApi.Func4`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**userName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFunc4Request struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**MainUpdateUserBody**](MainUpdateUserBody.md) |  | 

### Return type

[**MainUser**](MainUser.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## Func5

> MainUser Func5(ctx, userName).Execute()

delete user

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "./openapi"
)

func main() {
    userName := "userName_example" // string | 

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.UserApi.Func5(context.Background(), userName).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `UserApi.Func5``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `Func5`: MainUser
    fmt.Fprintf(os.Stdout, "Response from `UserApi.Func5`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**userName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFunc5Request struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**MainUser**](MainUser.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

