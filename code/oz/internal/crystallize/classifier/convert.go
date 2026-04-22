package classifier

import "github.com/joaoajmatos/oz/internal/crystallize/cache"

func classificationFromEntry(e cache.Entry) Classification {
	return Classification{
		Type:       ArtifactType(e.Type),
		Confidence: Confidence(e.Confidence),
		Title:      e.Title,
		Reason:     e.Reason,
		Source:     ClassifierSource(e.Source),
	}
}

func entryFromClassification(cl Classification) cache.Entry {
	return cache.Entry{
		Type:       string(cl.Type),
		Confidence: string(cl.Confidence),
		Title:      cl.Title,
		Reason:     cl.Reason,
		Source:     string(cl.Source),
	}
}
