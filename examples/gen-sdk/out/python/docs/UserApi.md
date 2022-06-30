# openapi_client.UserApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_user**](UserApi.md#create_user) | **POST** /api/user | create user
[**delete_user**](UserApi.md#delete_user) | **DELETE** /api/user/{user-name} | delete user
[**get_user**](UserApi.md#get_user) | **GET** /api/user/{user-name} | get user
[**get_users**](UserApi.md#get_users) | **GET** /api/user | search/list users
[**update_user**](UserApi.md#update_user) | **PUT** /api/user/{user-name} | update user


# **create_user**
> MainCreateUserOutput create_user(body)

create user

### Example


```python
import time
import openapi_client
from openapi_client.api import user_api
from openapi_client.model.main_create_user_output import MainCreateUserOutput
from openapi_client.model.main_create_user_input import MainCreateUserInput
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = openapi_client.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with openapi_client.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = user_api.UserApi(api_client)
    body = MainCreateUserInput(
        name="name_example",
        nick_name="nick_name_example",
        phone="phone_example",
    ) # MainCreateUserInput | 

    # example passing only required values which don't have defaults set
    try:
        # create user
        api_response = api_instance.create_user(body)
        pprint(api_response)
    except openapi_client.ApiException as e:
        print("Exception when calling UserApi->create_user: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**MainCreateUserInput**](MainCreateUserInput.md)|  |

### Return type

[**MainCreateUserOutput**](MainCreateUserOutput.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_user**
> MainUser delete_user(user_name)

delete user

### Example


```python
import time
import openapi_client
from openapi_client.api import user_api
from openapi_client.model.main_user import MainUser
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = openapi_client.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with openapi_client.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = user_api.UserApi(api_client)
    user_name = "user-name_example" # str | 

    # example passing only required values which don't have defaults set
    try:
        # delete user
        api_response = api_instance.delete_user(user_name)
        pprint(api_response)
    except openapi_client.ApiException as e:
        print("Exception when calling UserApi->delete_user: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **user_name** | **str**|  |

### Return type

[**MainUser**](MainUser.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_user**
> MainUser get_user(user_name)

get user

### Example


```python
import time
import openapi_client
from openapi_client.api import user_api
from openapi_client.model.main_user import MainUser
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = openapi_client.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with openapi_client.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = user_api.UserApi(api_client)
    user_name = "user-name_example" # str | 

    # example passing only required values which don't have defaults set
    try:
        # get user
        api_response = api_instance.get_user(user_name)
        pprint(api_response)
    except openapi_client.ApiException as e:
        print("Exception when calling UserApi->get_user: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **user_name** | **str**|  |

### Return type

[**MainUser**](MainUser.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_users**
> MainGetUsersOutput get_users()

search/list users

### Example


```python
import time
import openapi_client
from openapi_client.api import user_api
from openapi_client.model.main_get_users_output import MainGetUsersOutput
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = openapi_client.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with openapi_client.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = user_api.UserApi(api_client)
    query = "query_example" # str | query user (optional)
    count = True # bool | just response total count (optional)
    page_size = 1 # int | page size (optional)
    current_page = 1 # int | current page number, start at 1(defualt) (optional)
    sorter = "sorter_example" # str | column name (optional)
    order = "order_example" # str | asc(default)/desc (optional)

    # example passing only required values which don't have defaults set
    # and optional values
    try:
        # search/list users
        api_response = api_instance.get_users(query=query, count=count, page_size=page_size, current_page=current_page, sorter=sorter, order=order)
        pprint(api_response)
    except openapi_client.ApiException as e:
        print("Exception when calling UserApi->get_users: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **query** | **str**| query user | [optional]
 **count** | **bool**| just response total count | [optional]
 **page_size** | **int**| page size | [optional]
 **current_page** | **int**| current page number, start at 1(defualt) | [optional]
 **sorter** | **str**| column name | [optional]
 **order** | **str**| asc(default)/desc | [optional]

### Return type

[**MainGetUsersOutput**](MainGetUsersOutput.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **update_user**
> MainUser update_user(user_name, body)

update user

### Example


```python
import time
import openapi_client
from openapi_client.api import user_api
from openapi_client.model.main_update_user_body import MainUpdateUserBody
from openapi_client.model.main_user import MainUser
from pprint import pprint
# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = openapi_client.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with openapi_client.ApiClient() as api_client:
    # Create an instance of the API class
    api_instance = user_api.UserApi(api_client)
    user_name = "user-name_example" # str | 
    body = MainUpdateUserBody(
        nick_name="nick_name_example",
        phone="phone_example",
    ) # MainUpdateUserBody | 

    # example passing only required values which don't have defaults set
    try:
        # update user
        api_response = api_instance.update_user(user_name, body)
        pprint(api_response)
    except openapi_client.ApiException as e:
        print("Exception when calling UserApi->update_user: %s\n" % e)
```


### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **user_name** | **str**|  |
 **body** | [**MainUpdateUserBody**](MainUpdateUserBody.md)|  |

### Return type

[**MainUser**](MainUser.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

