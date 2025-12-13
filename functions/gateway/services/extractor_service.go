package services

// ExtractorRegistry manages all extractors
type ExtractorRegistry struct {
	extractors []EventExtractor
}

func NewExtractorRegistry() *ExtractorRegistry {
	return &ExtractorRegistry{
		extractors: []EventExtractor{
			&FacebookExtractor{},
		},
	}
}

func (r *ExtractorRegistry) GetExtractor(url string) EventExtractor {
	for _, extractor := range r.extractors {
		if extractor.CanHandle(url) {
			return extractor
		}
	}
	// Default to Not Known Urls
	return &GenericExtractor{}
}
