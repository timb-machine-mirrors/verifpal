/* SPDX-FileCopyrightText: © 2019-2020 Nadim Kobeissi <nadim@symbolic.software>
 * SPDX-License-Identifier: GPL-3.0-only */
// b3217fdc84899bc6c014cee70a82859d

package vplogic

func mutationMapInit(
	valKnowledgeMap KnowledgeMap, valPrincipalState PrincipalState, valAttackerState AttackerState, stage int,
) MutationMap {
	valMutationMap := MutationMap{
		Initialized:    true,
		Constants:      []Constant{},
		Mutations:      [][]Value{},
		Combination:    []Value{},
		DepthIndex:     []int{},
		OutOfMutations: false,
	}
	for _, v := range valAttackerState.Known {
		i := valueGetPrincipalStateIndexFromConstant(valPrincipalState, v.Constant)
		if mutationMapSkipValue(v, i, valKnowledgeMap, valPrincipalState, valAttackerState) {
			continue
		}
		a := valPrincipalState.BeforeMutate[i]
		c, r := mutationMapReplaceValue(
			a, v, i, stage,
			valPrincipalState, valAttackerState,
		)
		if len(r) == 0 {
			continue
		}
		aa := valueResolveValueInternalValuesFromPrincipalState(a, a, i, valPrincipalState, valAttackerState, true)
		r = append([]Value{aa}, r...)
		valMutationMap.Constants = append(valMutationMap.Constants, c)
		valMutationMap.Mutations = append(valMutationMap.Mutations, r)
	}
	valMutationMap.Combination = make([]Value, len(valMutationMap.Constants))
	valMutationMap.DepthIndex = make([]int, len(valMutationMap.Constants))
	for iiii := range valMutationMap.Constants {
		valMutationMap.DepthIndex[iiii] = 0
	}
	return valMutationMap
}

func mutationMapSkipValue(
	v Value, i int, valKnowledgeMap KnowledgeMap, valPrincipalState PrincipalState, valAttackerState AttackerState,
) bool {
	switch v.Kind {
	case "primitive":
		return true
	case "equation":
		return true
	}
	switch {
	case i < 0:
		return true
	case !strInSlice(valPrincipalState.Name, valPrincipalState.Wire[i]):
		return true
	case valPrincipalState.Guard[i]:
		if !strInSlice(valPrincipalState.Sender[i], valPrincipalState.MutatableTo[i]) {
			return true
		}
	case valPrincipalState.Creator[i] == valPrincipalState.Name:
		return true
	case !valPrincipalState.Known[i]:
		return true
	case !valueConstantIsUsedByPrincipalInKnowledgeMap(valKnowledgeMap, valPrincipalState.Name, v.Constant):
		return true
	case !intInSlice(valAttackerState.CurrentPhase, valPrincipalState.Phase[i]):
		return true
	}
	return false
}

func mutationMapReplaceValue(
	a Value, v Value, rootIndex int, stage int,
	valPrincipalState PrincipalState, valAttackerState AttackerState,
) (Constant, []Value) {
	switch a.Kind {
	case "constant":
		return v.Constant, mutationMapReplaceConstant(
			a, stage, valPrincipalState, valAttackerState,
		)
	case "primitive":
		return v.Constant, mutationMapReplacePrimitive(
			a, rootIndex, stage, valPrincipalState, valAttackerState,
		)
	case "equation":
		return v.Constant, mutationMapReplaceEquation(
			a, stage, valAttackerState,
		)
	}
	return v.Constant, []Value{}
}

func mutationMapReplaceConstant(
	a Value, stage int,
	valPrincipalState PrincipalState, valAttackerState AttackerState,
) []Value {
	mutations := []Value{}
	if valueIsGOrNil(a.Constant) {
		return mutations
	}
	mutations = append(mutations, valueN)
	if stage <= 3 {
		return mutations
	}
	for _, v := range valAttackerState.Known {
		switch v.Kind {
		case "constant":
			if valueIsGOrNil(v.Constant) {
				continue
			}
			c := valueResolveConstant(v.Constant, valPrincipalState)
			switch c.Kind {
			case "constant":
				if valueEquivalentValueInValues(c, mutations) < 0 {
					mutations = append(mutations, c)
				}
			}
		}
	}
	return mutations
}

func mutationMapReplacePrimitive(
	a Value, rootIndex int, stage int,
	valPrincipalState PrincipalState, valAttackerState AttackerState,
) []Value {
	mutations := []Value{}
	for _, v := range valAttackerState.Known {
		switch v.Kind {
		case "primitive":
			a = valueResolveValueInternalValuesFromPrincipalState(
				a, a, rootIndex, valPrincipalState, valAttackerState, false,
			)
			if !injectMatchSkeletons(v.Primitive, injectPrimitiveSkeleton(a.Primitive)) {
				continue
			}
			if valueEquivalentValueInValues(v, mutations) < 0 {
				mutations = append(mutations, v)
			}
		}
	}
	injectants := inject(
		a.Primitive, a.Primitive, true,
		valPrincipalState, valAttackerState, stage,
	)
	for _, aa := range injectants {
		if valueEquivalentValueInValues(aa, mutations) < 0 {
			mutations = append(mutations, aa)
		}
	}
	return mutations
}

func mutationMapReplaceEquation(a Value, stage int, valAttackerState AttackerState) []Value {
	mutations := []Value{}
	if stage <= 3 {
		return []Value{valueGN}
	}
	for _, v := range valAttackerState.Known {
		switch v.Kind {
		case "equation":
			switch len(v.Equation.Values) {
			case len(a.Equation.Values):
				if valueEquivalentValueInValues(v, mutations) < 0 {
					mutations = append(mutations, v)
				}
			}
		}
	}
	return mutations
}

func mutationMapNext(valMutationMap MutationMap) MutationMap {
	if len(valMutationMap.Combination) == 0 {
		valMutationMap.OutOfMutations = true
		return valMutationMap
	}
	for i := 0; i < len(valMutationMap.Combination); i++ {
		valMutationMap.Combination[i] = valMutationMap.Mutations[i][valMutationMap.DepthIndex[i]]
		if i != len(valMutationMap.Combination)-1 {
			continue
		}
		valMutationMap.DepthIndex[i] = valMutationMap.DepthIndex[i] + 1
		valMutationMap.LastIncrement = i
		for ii := i; ii >= 0; ii-- {
			if valMutationMap.DepthIndex[ii] != len(valMutationMap.Mutations[ii]) {
				continue
			}
			if ii <= 0 {
				valMutationMap.OutOfMutations = true
				break
			}
			valMutationMap.DepthIndex[ii] = 0
			valMutationMap.DepthIndex[ii-1] = valMutationMap.DepthIndex[ii-1] + 1
			valMutationMap.LastIncrement = ii - 1
		}
	}
	return valMutationMap
}
