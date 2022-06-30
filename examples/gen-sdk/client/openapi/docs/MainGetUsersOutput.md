# MainGetUsersOutput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**List** | [**[]MainUser**](MainUser.md) |  | 
**Total** | **int32** |  | 

## Methods

### NewMainGetUsersOutput

`func NewMainGetUsersOutput(list []MainUser, total int32, ) *MainGetUsersOutput`

NewMainGetUsersOutput instantiates a new MainGetUsersOutput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMainGetUsersOutputWithDefaults

`func NewMainGetUsersOutputWithDefaults() *MainGetUsersOutput`

NewMainGetUsersOutputWithDefaults instantiates a new MainGetUsersOutput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetList

`func (o *MainGetUsersOutput) GetList() []MainUser`

GetList returns the List field if non-nil, zero value otherwise.

### GetListOk

`func (o *MainGetUsersOutput) GetListOk() (*[]MainUser, bool)`

GetListOk returns a tuple with the List field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetList

`func (o *MainGetUsersOutput) SetList(v []MainUser)`

SetList sets List field to given value.


### GetTotal

`func (o *MainGetUsersOutput) GetTotal() int32`

GetTotal returns the Total field if non-nil, zero value otherwise.

### GetTotalOk

`func (o *MainGetUsersOutput) GetTotalOk() (*int32, bool)`

GetTotalOk returns a tuple with the Total field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotal

`func (o *MainGetUsersOutput) SetTotal(v int32)`

SetTotal sets Total field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


