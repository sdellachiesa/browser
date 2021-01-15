// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package ql provides an API for building InfluxQL queries.
//
// Currently it only supports 'SELECT' and and 'SHOW TAG VALUES' queries.
//
// The builders will not ensure that the query returned will be a valid one in
// terms of completeness. The user of the package is responsible to compose a
// full query with the builder.
//
// An example of a SELECT query:
//
//  ql.Select("a", "b").From("c").Where(ql.Eq(ql.And(), "a", "d", "g"))
//
// Will return:
//
//  SELECT a, b FROM c WHERE a='d' AND a='g'
//
package ql

import (
	"bytes"
	"fmt"
	"time"
)

// Querier interface provides the Query method.
type Querier interface {
	Query() (string, []interface{})
}

// The QueryFunc type is an adapter to allow the use of ordinary functions as
// Querier.
type QueryFunc func() (string, []interface{})

// Query calls qf()
func (qf QueryFunc) Query() (string, []interface{}) {
	return qf()
}

// Builder is the base builder for an Influx QL query.
type Builder struct {
	bytes.Buffer
	args []interface{}
}

func (b Builder) Query() (string, []interface{}) {
	return b.String(), b.args
}

// Append appens the given string to the query if it is a valid identifier.
func (b *Builder) Append(s string) *Builder {
	switch {
	case len(s) == 0:
	default:
		b.WriteString(s)
	}

	return b
}

// AppendWithQuotes appens the given string to the query builder with double
// quotes.
func (b *Builder) AppendWithQuotes(s string) *Builder {
	fmt.Fprintf(b, "%q", s)
	return b
}

// AppendWithComma appens the given strings and separates them with a comma.
func (b *Builder) AppendWithComma(s ...string) *Builder {
	for i := range s {
		if i > 0 {
			b.WriteString(", ")
		}
		b.Append(s[i])
	}

	return b
}

// AppendWithQuotesAndComma appens the given strings with double quotes and
// separates them with a comma.
func (b *Builder) AppendWithQuotesAndComma(s ...string) *Builder {
	for i := range s {
		s[i] = fmt.Sprintf("%q", s[i])
	}

	return b.AppendWithComma(s...)
}

// merge merges the given Querier with the current builder.
func (b *Builder) merge(q Querier) *Builder {
	s, args := q.Query()
	b.args = append(b.args, args)
	b.Append(s)
	return b
}

type Operator string

const (
	EQ      Operator = "="
	NEQ     Operator = "!="
	MATCH   Operator = "=~"
	NOMATCH Operator = "!~"
)

// ShowTagValuesBuilder is a builder for a 'SHOW TAG VALUES' query.
type ShowTagValuesBuilder struct {
	b     Builder
	from  []string
	in    []string
	where *WhereBuilder
}

// ShowTagValues returns the base for building a 'SHOW TAG VALUES'
// query.
func ShowTagValues() *ShowTagValuesBuilder {
	return &ShowTagValuesBuilder{}
}

func (st *ShowTagValuesBuilder) From(f ...string) *ShowTagValuesBuilder {
	if len(f) < 1 {
		f = []string{"/.*/"}
	}
	st.from = f

	return st
}

func (st *ShowTagValuesBuilder) WithKeyIn(tagKeys ...string) *ShowTagValuesBuilder {
	st.in = tagKeys
	return st
}

func (st *ShowTagValuesBuilder) Where(q ...Querier) *ShowTagValuesBuilder {
	if len(q) > 0 {
		st.where = Where(q...)
	}
	return st
}

func (st *ShowTagValuesBuilder) Query() (string, []interface{}) {
	st.b.WriteString("SHOW TAG VALUES ")

	if len(st.from) > 0 {
		st.b.Append("FROM ")
		st.b.AppendWithComma(st.from...)
	}

	if len(st.in) > 0 {
		st.b.Append(" WITH KEY IN (")
		st.b.AppendWithQuotesAndComma(st.in...)
		st.b.Append(")")
	}

	if st.where != nil {
		w, _ := st.where.Query()
		if len(w) > 0 {
			st.b.Append(" WHERE ")
			st.b.Append(w)
		}

	}

	return st.b.String(), nil
}

// ShowMeasurementBuilder is a builder for a 'SHOW MEASUREMENTS' query.
type ShowMeasurementBuilder struct {
	b        Builder
	operator Operator
	regex    string
	where    *WhereBuilder
}

// ShowMeasurement returns the base for building a 'SHOW MEASUREMENT' query.
func ShowMeasurement() *ShowMeasurementBuilder {
	return &ShowMeasurementBuilder{}
}

func (sm *ShowMeasurementBuilder) With(o Operator, regex string) *ShowMeasurementBuilder {
	sm.operator = o
	sm.regex = regex

	return sm
}

func (sm *ShowMeasurementBuilder) Where(q ...Querier) *ShowMeasurementBuilder {
	if len(q) > 0 {
		sm.where = Where(q...)
	}
	return sm
}

func (sm *ShowMeasurementBuilder) Query() (string, []interface{}) {
	sm.b.WriteString("SHOW MEASUREMENTS")

	if sm.operator != "" && sm.regex != "" {
		sm.b.Append(" WITH MEASUREMENT ")
		sm.b.Append(string(sm.operator))
		sm.b.Append(" /" + sm.regex + "/")
	}

	if sm.where != nil {
		w, _ := sm.where.Query()
		if len(w) > 0 {
			sm.b.Append(" WHERE ")
			sm.b.Append(w)
		}
	}

	return sm.b.String(), nil
}

// SelectBuilder is a builder for a 'SELECT' query.
type SelectBuilder struct {
	b        Builder
	columns  []string
	from     []string
	where    *WhereBuilder
	order    string
	group    string
	orderDir string
	limit    string
	timezone string
}

// Select returns the base for building a 'SELECT' query.
func Select(columns ...string) *SelectBuilder {
	if len(columns) == 0 {
		columns = []string{"*"}
	}
	return &SelectBuilder{columns: columns}
}

func (sb *SelectBuilder) From(f ...string) *SelectBuilder {
	if len(f) < 1 {
		f = []string{"/.*/"}
	}
	sb.from = f

	return sb
}

func (sb *SelectBuilder) Where(q ...Querier) *SelectBuilder {
	if len(q) > 0 {
		sb.where = Where(q...)
	}
	return sb
}

func (sb *SelectBuilder) OrderBy(column string) *SelectBuilder {
	sb.order = column
	return sb
}

func (sb *SelectBuilder) GroupBy(column string) *SelectBuilder {
	sb.group = column
	return sb
}

func (sb *SelectBuilder) ASC() *SelectBuilder {
	sb.orderDir = " ASC"
	return sb
}

func (sb *SelectBuilder) Limit(l int64) *SelectBuilder {
	if l > 0 {
		sb.limit = fmt.Sprintf(" LIMIT %d", l)
	}
	return sb
}

func (sb *SelectBuilder) TZ(tz string) *SelectBuilder {
	sb.timezone = fmt.Sprintf(" TZ('%s')", tz)
	return sb
}

func (sb *SelectBuilder) Query() (string, []interface{}) {
	sb.b.WriteString("SELECT ")

	sb.b.AppendWithComma(sb.columns...)

	if len(sb.from) > 0 {
		sb.b.Append(" FROM ")
		sb.b.AppendWithComma(sb.from...)
	}

	if sb.where != nil {
		sb.b.Append(" WHERE ")
		sb.b.merge(sb.where)
	}

	if sb.group != "" {
		sb.b.Append(" GROUP BY ")
		sb.b.Append(sb.group)
	}

	if sb.order != "" {
		sb.b.Append(" ORDER BY ")
		sb.b.Append(sb.order)
	}

	if sb.orderDir != "" {
		sb.b.Append(sb.orderDir)
	}

	if sb.limit != "" {
		sb.b.Append(sb.limit)
	}

	if sb.timezone != "" {
		sb.b.Append(sb.timezone)
	}

	return sb.b.String(), sb.b.args
}

// WhereBuilder is a builder for the 'WHERE' clause of a query.
type WhereBuilder struct {
	b       Builder
	queries []Querier
}

func Where(q ...Querier) *WhereBuilder {
	return &WhereBuilder{queries: q}
}

func (wb *WhereBuilder) Query() (string, []interface{}) {
	for _, query := range wb.queries {
		if query == nil {
			continue
		}

		q, _ := query.Query()

		if len(wb.b.String()) == 0 {
			// If the buffer is empty and the next query
			// is an operator skip it.
			_, ok := query.(*OperatorBuilder)
			if ok {
				continue
			}
		}

		wb.b.Append(q)
	}

	return wb.b.String(), nil
}

// OperatorBuilder is a builder for combining WHERE clauses
// with AND and OR operators.
type OperatorBuilder struct {
	b Builder
}

func And() *OperatorBuilder {
	o := &OperatorBuilder{}
	o.b.Append(" AND ")
	return o
}

func Or() *OperatorBuilder {
	o := &OperatorBuilder{}
	o.b.Append(" OR ")
	return o
}

func (o *OperatorBuilder) Query() (string, []interface{}) {
	return o.b.String(), o.b.args
}

// Eq returns a query part which compares column to each given value, joining
// them together with the given OperatorBuilder.
//
//   Eq(And(), "a", "b", "c") -> a='b' AND a='c'
func Eq(join *OperatorBuilder, column string, values ...string) Querier {
	return QueryFunc(func() (string, []interface{}) {
		return comp(join, "=", column, values...), nil
	})
}

func Lte(join *OperatorBuilder, column string, values ...string) Querier {
	return QueryFunc(func() (string, []interface{}) {
		return comp(join, "<=", column, values...), nil
	})
}

func Gte(join *OperatorBuilder, column string, values ...string) Querier {
	return QueryFunc(func() (string, []interface{}) {
		return comp(join, ">=", column, values...), nil
	})
}

func comp(join *OperatorBuilder, operator, column string, values ...string) string {
	var b Builder

	for i, v := range values {
		if len(v) == 0 {
			continue
		}

		if i > 0 && len(b.String()) > 0 {
			b.merge(join)
		}
		fmt.Fprintf(&b, "%s%s'%s'", column, operator, v)
	}

	return b.String()
}

func TimeRange(from, to time.Time) Querier {
	var b Builder
	return QueryFunc(func() (string, []interface{}) {
		fmt.Fprintf(&b, "time >= '%s' AND time <= '%s'",
			from.Format("2006-01-02T15:04:05Z"),
			to.Format("2006-01-02T15:04:05Z"),
		)
		return b.String(), nil
	})
}
