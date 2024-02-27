package pagination

import (
	"math"
	"strconv"

	"github.com/HannesOberreiter/gbif-extinct/internal"
)

type Pages struct {
	CURRENT  int
	NEXT     int
	PREVIOUS int
	LAST     int
}

type PagesString struct {
	CURRENT  string
	NEXT     string
	PREVIOUS string
	LAST     string
}

func CalculatePages(counts internal.Counts, payload internal.Payload) PagesString {
	var response Pages
	response.CURRENT = 1

	if payload.PAGE != nil {
		page, err := strconv.Atoi(*payload.PAGE)
		if err == nil {
			response.CURRENT = page
		}
	}

	response.LAST = int(math.Ceil(float64(counts.TaxaCount) / float64(internal.PageLimit)))
	if response.CURRENT == response.LAST {
		response.NEXT = response.LAST
		response.PREVIOUS = response.LAST - 1
	} else if response.CURRENT == 1 {
		response.NEXT = response.CURRENT + 1
		response.PREVIOUS = response.CURRENT
	} else {
		response.NEXT = response.CURRENT + 1
		response.PREVIOUS = response.CURRENT - 1
	}

	return PagesString{
		CURRENT:  strconv.Itoa(response.CURRENT),
		NEXT:     strconv.Itoa(response.NEXT),
		PREVIOUS: strconv.Itoa(response.PREVIOUS),
		LAST:     strconv.Itoa(response.LAST),
	}
}
