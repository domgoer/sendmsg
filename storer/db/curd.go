/* ====================================================
#   Copyright (C)2019 All rights reserved.
#
#   Author        : domchan
#   Email         : 814172254@qq.com
#   File Name     : curd.go
#   Created       : 2019/1/21 14:40
#   Last Modified : 2019/1/21 14:40
#   Describe      :
#
# ====================================================*/
package db

import (
	"context"
	"database/sql"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/hiruok/msg-pusher/storer"
	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// query 获取单条数据
func query(ctx context.Context, out interface{}, typeN, sql string, args ...interface{}) error {
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		parentCtx := parentSpan.Context()
		span := opentracing.StartSpan(typeN, opentracing.ChildOf(parentCtx))
		ext.SpanKindRPCClient.Set(span)
		ext.PeerService.Set(span, "mysql")
		span.SetTag("sql.query", sql)
		span.SetTag("sql.param", args)
		defer span.Finish()
		ctx = opentracing.ContextWithSpan(ctx, span)
	}

	return storer.DB.GetContext(ctx, out, sql, args...)
}

// list 获取集合数据
func list(ctx context.Context, out interface{}, typeN, sql string, args ...interface{}) error {
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		parentCtx := parentSpan.Context()
		span := opentracing.StartSpan(typeN, opentracing.ChildOf(parentCtx))
		ext.SpanKindRPCClient.Set(span)
		ext.PeerService.Set(span, "mysql")
		span.SetTag("sql.query", sql)
		span.SetTag("sql.param", args)
		defer span.Finish()
		ctx = opentracing.ContextWithSpan(ctx, span)
	}
	return storer.DB.SelectContext(ctx, out, sql, args...)
}

// update 更新数据
func update(ctx context.Context, typeN, sqlStr string, args ...interface{}) (*sqlx.Tx, error) {
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		parentCtx := parentSpan.Context()
		span := opentracing.StartSpan(typeN, opentracing.ChildOf(parentCtx))
		ext.SpanKindRPCClient.Set(span)
		ext.PeerService.Set(span, "mysql")
		span.SetTag("sql.update", sqlStr)
		span.SetTag("sql.param", args)
		defer span.Finish()
		ctx = opentracing.ContextWithSpan(ctx, span)
	}

	tx, err := storer.DB.Beginx()
	if err != nil {
		return nil, err
	}
	stmt, err := tx.PreparexContext(ctx, sqlStr)
	if err != nil {
		return tx, err
	}
	defer stmt.Close()
	var res sql.Result
	res, err = stmt.ExecContext(ctx, args...)
	if err != nil {
		return tx, err
	}
	if i, _ := res.RowsAffected(); i == 0 {
		return tx, ErrNoRowsEffected
	}
	return tx, nil
}

func insert(ctx context.Context, typeN, sqlStr string, args ...interface{}) (tx *sqlx.Tx, err error) {
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		parentCtx := parentSpan.Context()
		span := opentracing.StartSpan(typeN, opentracing.ChildOf(parentCtx))
		ext.SpanKindRPCClient.Set(span)
		ext.PeerService.Set(span, "mysql")
		span.SetTag("sql.insert", sqlStr)
		span.SetTag("sql.param", args)
		defer span.Finish()
		ctx = opentracing.ContextWithSpan(ctx, span)
	}

	tx, err = storer.DB.Beginx()
	if err != nil {
		return nil, err
	}
	stmt, err := tx.PreparexContext(ctx, sqlStr)
	if err != nil {
		return tx, err
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, args...)
	if err != nil && err.(*mysql.MySQLError).Number == 1062 {
		err = ErrUniqueKeyExsits
	}
	return tx, err
}

func batch(ctx context.Context, table string, params []string, args ...interface{}) error {
	pn := len(params)
	if pn == 0 {
		return nil
	}
	n := len(args) / pn

	var sb strings.Builder
	sb.WriteString("INSERT INTO ")
	sb.WriteString(table)
	sb.WriteString(" (")
	sb.WriteString(strings.Join(params, ","))
	sb.WriteString(") VALUES ")
	// sql : INSERT INTO sms (l1,l2,l3...) VALUES

	var ws []string
	var values []string
	for i := 0; i < pn; i++ {
		ws = append(ws, "?")
		values = append(values, params[i]+"=IF(version<=VALUES(version),VALUES("+params[i]+"),"+params[i]+")")
	}
	pl := "(" + strings.Join(ws, ",") + ")"
	vs := strings.Join(values, ",")

	for i := 0; i < n; i++ {
		sb.WriteString(pl)
		if i != n-1 {
			sb.WriteString(",")
		}
	}
	sb.WriteString(" ON DUPLICATE KEY UPDATE ")
	sb.WriteString(vs)
	sql := sb.String()
	_, err := storer.DB.ExecContext(ctx, sql, args...)
	if err != nil {
		return err
	}
	return err

}
