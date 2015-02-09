package sorting_network

import (
	"github.com/vale1410/bule/constraints"
	"github.com/vale1410/bule/sat"
	"github.com/vale1410/bule/sorters"
)

type Translation struct {
	PB     constraints.Threshold
	Tare   int64
	Sorter sorters.Sorter
	Bags   [][]sat.Literal
	typ    sorters.SortingNetworkType
}

func NewTranslation(pb constraints.Threshold) (t Translation) {
	t.PB = pb
	t.typ = sorters.OddEven
	return
}

func (t *Translation) CreateSorter() {

	total := t.PB.NormalizeAtMost()

	if total <= t.PB.K {
		panic("sum of weights is too low to make a difference!")
	}
	if t.PB.K == 0 {
		panic("Threshold is 0 with positive weights. All literals are facts!")
	}

	t.CreateBags()

	layers := make([]sorters.Sorter, len(t.Bags))

	for i, bag := range t.Bags {

		layers[i] = sorters.CreateSortingNetwork(len(bag), -1, t.typ)

		t.PB.LitIn = append(t.PB.LitIn, bag...) // this might have to be reversed
	}

	t.Sorter.In = make([]int, 0, len(t.PB.LitIn))
	t.Sorter.Out = make([]int, 0, len(t.PB.LitIn))

	offset := 2

	// determine the constant and what to add on both sides
	layerPow2 := int64(1 << uint(len(t.Bags)))

	tare := layerPow2 - ((t.PB.K + 1) % layerPow2)
	tare = tare % layerPow2
	t.Tare = tare
	bTare := constraints.Binary(tare)

	// output of sorter in layer $i-1$
	bIn := make([]int, 0)

	finalMapping := make(map[int]int, len(t.Sorter.In))

	for i, layer := range layers {

		offset = layer.Normalize(offset, []int{})
		t.Sorter.Comparators = append(t.Sorter.Comparators, layer.Comparators...)

		t.Sorter.In = append(t.Sorter.In, layer.In...)

		size := len(bIn) + len(layers[i].In)

		mergeIn := make([]int, 0, size)
		mergeIn = append(mergeIn, bIn...)
		mergeIn = append(mergeIn, layer.Out...)

		merger := sorters.CreateSortingNetwork(size, len(bIn), t.typ)
		offset = merger.Normalize(offset, mergeIn)

		// halving circuit:

		odd := 1

		if i < len(bTare) && bTare[i] == 1 {
			odd = 0
			bIn = make([]int, (len(merger.Out)+1)/2)
		} else {
			bIn = make([]int, len(merger.Out)/2)
		}

		// Alternate depending on bTare
		for j, x := range merger.Out {
			if j%2 == odd {
				bIn[j/2] = x
			} else if i < len(layers)-1 { // not in last layer, but else
				finalMapping[x] = -1
			}
		}

		t.Sorter.Comparators = append(t.Sorter.Comparators, merger.Comparators...)

	}

	// outLastLayer identifies the nth output in the last layer
	outLastLayer := ((t.PB.K + 1 + tare) / int64(layerPow2)) - 1
	idSetToZero := bIn[outLastLayer]

	// and propagate the rest backwards
	setTo := -1
	for _, id := range t.Sorter.ComputeOut() {
		if id == idSetToZero {
			setTo = 0
		}
		if _, ok := finalMapping[id]; !ok {
			finalMapping[id] = setTo
		}
	}

	t.Sorter.PropagateBackwards(finalMapping)
	t.Sorter.Normalize(2, []int{})

	//fmt.Println("final debug: tSorter", t.Sorter)
}

func (t *Translation) CreateBags() {

	if !t.PB.IsNormalized() {
		t.PB.Print10()
		panic("Threshold is not normalized before creating bags")
	}

	nBags := len(constraints.Binary(t.PB.K))
	bins := make([][]int, len(t.PB.Entries))
	bagPos := make([]int, nBags)
	bagSize := make([]int, nBags)

	maxWeight := int64(0)

	for i, e := range t.PB.Entries {
		bins[i] = constraints.Binary(e.Weight)

		for j, x := range bins[i] {
			bagSize[j] += x
		}

		if maxWeight < e.Weight {
			maxWeight = e.Weight
		}

	}

	t.Bags = make([][]sat.Literal, len(constraints.Binary(maxWeight)))

	for i, _ := range t.Bags {
		t.Bags[i] = make([]sat.Literal, bagSize[i])
	}

	for i, e := range t.PB.Entries {
		for j, x := range bins[i] {
			if x == 1 {
				t.Bags[j][bagPos[j]] = e.Literal
				bagPos[j]++
			}
		}
	}
}
