package authenticator

func AudiencesAreAcceptable(apiAuds, responseAudiences Audiences) bool {
	if len(apiAuds) == 0 || len(responseAudiences) == 0 {
		return true
	}

	return len(apiAuds.Intersect(responseAudiences)) > 0
}
