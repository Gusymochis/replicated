# Body1

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ActivationEmail** | **string** | If activation is required this is the email the code will be sent to. | [default to null]
**AppId** | **string** | App Id that this license will be associated with. | [default to null]
**Assignee** | **string** | License Label name, ie name of customer who is using license. | [default to null]
**ChannelId** | **string** | Channel id that the license will be associated with (deprecated, use \&quot;channels\&quot;). | [default to null]
**Channels** | [***LicenseChannels**](LicenseChannels.md) |  | [optional] [default to null]
**ConsoleAuthOptions** | **[]string** |  | [optional] [default to null]
**EnabledFeatures** | [**map[string]interface{}**](interface{}.md) |  | [optional] [default to null]
**ExpirationDate** | **string** | Date that the license will expire, can be null for no expiration or formatted by year-month-day ex. 2016-02-02. | [default to null]
**ExpirationPolicy** | **string** | Defines expiration policy for this license.  Values: ignore: replicated will take no action on a expired license noupdate-norestart: application updates will not be downloaded, and once the application is stopped, it will not be started again noupdate-stop: application updates will not be downloaded and the application will be stopped | [default to null]
**FieldValues** | [***LicenseFieldValues**](LicenseFieldValues.md) |  | [default to null]
**LicenseType** | **string** | LicenseType can be set to \&quot;dev\&quot;, \&quot;trial\&quot;, or \&quot;prod\&quot; | [default to null]
**RequireActivation** | **bool** | If this license requires activation set to true, make sure to set activation email as well. | [default to null]
**UpdatePolicy** | **string** | If set to automatic will auto update remote license installation with every release. If set to manual will update only when on-premise admin clicks the install update button. | [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


