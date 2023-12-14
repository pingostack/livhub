package rtsp

type Status int

/*
100 - Continue
200 - OK
201 - Created
250 - Low on Storage Space
300 - Multiple Choices
301 - Moved Permanently
302 - Moved Temporarily
303 - See Other
304 - Not Modified
305 - Use Proxy
400 - Bad Request
401 - Unauthorized
402 - Payment Required
403 - Forbidden
404 - Not Found
405 - Method Not Allowed
406 - Not Acceptable
407 - Proxy Authentication Required
408 - Request Time-out
410 - Gone
411 - Length Required
412 - Precondition Failed
413 - Request Entity Too Large
414 - Request-URI Too Large
415 - Unsupported Media Type
451 - Parameter Not Understood
452 - Conference Not Found
453 - Not Enough Bandwidth
454 - Session Not Found
455 - Method Not Valid in This State
456 - Header Field Not Valid for Resource
457 - Invalid Range
458 - Parameter Is Read-Only
459 - Aggregate operation not allowed
460 - Only aggregate operation allowed
461 - Unsupported transport
462 - Destination unreachable
463 - Key management Failure
500 - Internal Server Error
501 - Not Implemented
502 - Bad Gateway
503 - Service Unavailable
504 - Gateway Time-out
505 - RTSP Version not supported
551 - Option not supported
*/
const (
	StatusContinue               Status = 100
	StatusOK                     Status = 200
	StatusCreated                Status = 201
	StatusLowOnStorageSpace      Status = 250
	StatusMultipleChoices        Status = 300
	StatusMovedPermanently       Status = 301
	StatusMovedTemporarily       Status = 302
	StatusSeeOther               Status = 303
	StatusNotModified            Status = 304
	StatusUseProxy               Status = 305
	StatusBadRequest             Status = 400
	StatusUnauthorized           Status = 401
	StatusPaymentRequired        Status = 402
	StatusForbidden              Status = 403
	StatusNotFound               Status = 404
	StatusMethodNotAllowed       Status = 405
	StatusNotAcceptable          Status = 406
	StatusProxyAuthRequired      Status = 407
	StatusRequestTimeout         Status = 408
	StatusGone                   Status = 410
	StatusLengthRequired         Status = 411
	StatusPreconditionFailed     Status = 412
	StatusRequestEntityTooLarge  Status = 413
	StatusRequestURITooLarge     Status = 414
	StatusUnsupportedMediaType   Status = 415
	StatusParameterNotUnderstood Status = 451
	StatusConferenceNotFound     Status = 452
	StatusNotEnoughBandwidth     Status = 453
	StatusSessionNotFound        Status = 454
	StatusMethodNotValid         Status = 455
	StatusHeaderFieldNotValid    Status = 456
	StatusInvalidRange           Status = 457
	StatusParameterIsReadOnly    Status = 458
	StatusAggregateNotAllowed    Status = 459
	StatusOnlyAggregateAllowed   Status = 460
	StatusUnsupportedTransport   Status = 461
	StatusDestinationUnreachable Status = 462
	StatusKeyManagementFailure   Status = 463
	StatusInternalServerError    Status = 500
	StatusNotImplemented         Status = 501
	StatusBadGateway             Status = 502
	StatusServiceUnavailable     Status = 503
	StatusGatewayTimeout         Status = 504
	StatusVersionNotSupported    Status = 505
	StatusOptionNotSupported     Status = 551
)

func (status Status) String() string {
	switch status {
	case StatusContinue:
		return "Continue"
	case StatusOK:
		return "OK"
	case StatusCreated:
		return "Created"
	case StatusLowOnStorageSpace:
		return "Low on Storage Space"
	case StatusMultipleChoices:
		return "Multiple Choices"
	case StatusMovedPermanently:
		return "Moved Permanently"
	case StatusMovedTemporarily:
		return "Moved Temporarily"
	case StatusSeeOther:
		return "See Other"
	case StatusNotModified:
		return "Not Modified"
	case StatusUseProxy:
		return "Use Proxy"
	case StatusBadRequest:
		return "Bad Request"
	case StatusUnauthorized:
		return "Unauthorized"
	case StatusPaymentRequired:
		return "Payment Required"
	case StatusForbidden:
		return "Forbidden"
	case StatusNotFound:
		return "Not Found"
	case StatusMethodNotAllowed:
		return "Method Not Allowed"
	case StatusNotAcceptable:
		return "Not Acceptable"
	case StatusProxyAuthRequired:
		return "Proxy Authentication Required"
	case StatusRequestTimeout:
		return "Request Time-out"
	case StatusGone:
		return "Gone"
	case StatusLengthRequired:
		return "Length Required"
	case StatusPreconditionFailed:
		return "Precondition Failed"
	case StatusRequestEntityTooLarge:
		return "Request Entity Too Large"
	case StatusRequestURITooLarge:
		return "Request-URI Too Large"
	case StatusUnsupportedMediaType:
		return "Unsupported Media Type"
	case StatusParameterNotUnderstood:
		return "Parameter Not Understood"
	case StatusConferenceNotFound:
		return "Conference Not Found"
	case StatusNotEnoughBandwidth:
		return "Not Enough Bandwidth"
	case StatusSessionNotFound:
		return "Session Not Found"
	case StatusMethodNotValid:
		return "Method Not Valid in This State"
	case StatusHeaderFieldNotValid:
		return "Header Field Not Valid for Resource"
	case StatusInvalidRange:
		return "Invalid Range"
	case StatusParameterIsReadOnly:
		return "Parameter Is Read-Only"
	case StatusAggregateNotAllowed:
		return "Aggregate operation not allowed"
	case StatusOnlyAggregateAllowed:
		return "Only aggregate operation allowed"
	case StatusUnsupportedTransport:
		return "Unsupported transport"
	case StatusDestinationUnreachable:
		return "Destination unreachable"
	case StatusKeyManagementFailure:
		return "Key management Failure"
	case StatusInternalServerError:
		return "Internal Server Error"
	case StatusNotImplemented:
		return "Not Implemented"
	case StatusBadGateway:
		return "Bad Gateway"
	case StatusServiceUnavailable:
		return "Service Unavailable"
	case StatusGatewayTimeout:
		return "Gateway Time-out"
	case StatusVersionNotSupported:
		return "RTSP Version not supported"
	case StatusOptionNotSupported:
		return "Option not supported"
	default:
		return "Unknown"
	}
}
