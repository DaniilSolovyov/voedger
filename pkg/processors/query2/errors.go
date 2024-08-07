/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 *
 * @author Daniil Solovyov
 */

package query2

import "errors"

var (
	errEmptyConstraint             = errors.New("empty constraint")
	errTooMuchConstraints          = errors.New("too much constraints")
	errAbsentConstraintContainedIn = errors.New("absent constraint 'ContainedIn'")
)
