package janus

func String(v string) *string {
	return &v
}

func Int64(v int64) *int64 {
	return &v
}

func Int(v int) *int {
	return &v
}

func Bool(v bool) *bool {
	return &v
}

func CreateGenericRequest(requestType *string, _type *string, room *int64, id *int, secret *string, permanent *bool) *GenericRequest {
	return &GenericRequest{
		Request:   requestType,
		Type:      _type,
		Room:      room,
		Id:        id,
		Secret:    secret,
		Permanent: permanent,
	}
}

func CreateExistsRequest(room int64) *GenericRequest {
	return CreateGenericRequest(String(Exists), nil, Int64(room), nil, nil, nil)
}

func CreateDestroyRoomIdRequest(room int64, secret string, permanent bool) *GenericRequest {
	return CreateGenericRequest(String(Destroy), nil, Int64(room), nil, String(secret), Bool(permanent))
}

func CreateDestroyIdRequest(id int, secret string) *GenericRequest {
	return CreateGenericRequest(String(Destroy), nil, nil, Int(id), String(secret), nil)
}

func CreateListParticipantsRequest(room int64) *GenericRequest {
	return CreateGenericRequest(String(ListParticipants), nil, Int64(room), nil, nil, nil)
}

func CreateInfoRequest(id int, secret string) *GenericRequest {
	return CreateGenericRequest(String(Info), nil, nil, Int(id), String(secret), nil)
}
