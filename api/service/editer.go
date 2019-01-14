/* ====================================================
#   Copyright (C)2019 All rights reserved.
#
#   Author        : domchan
#   Email         : 814172254@qq.com
#   File Name     : editer.go
#   Created       : 2019/1/14 15:33
#   Last Modified : 2019/1/14 15:33
#   Describe      :
#
# ====================================================*/
package service

import (
	"context"
	"github.com/sirupsen/logrus"
	"uuabc.com/sendmsg/api/model"
	"uuabc.com/sendmsg/api/storer/cache"
	"uuabc.com/sendmsg/api/storer/db"
	cache2 "uuabc.com/sendmsg/pkg/cache"
	"uuabc.com/sendmsg/pkg/errors"
	"uuabc.com/sendmsg/pkg/pb/meta"
)

var EditerImpl editerImpl

type editerImpl struct{}

// TODO send to mq
func (e editerImpl) Edit(ctx context.Context, m Meta) error {
	m.Transfer(false)
	err := e.edit(ctx, m)
	if err == nil {
		return nil
	}
	// 转换err类型
	if _, ok := err.(*errors.Error); !ok {
		if err == cache2.ErrCacheMiss {
			err = errors.ErrMsgNotFound
		} else {
			logrus.WithFields(logrus.Fields{
				"method": "Edit",
				"error":  err,
			}).Errorf("数据操作异常")
			err = errors.NewError(
				10000000,
				err.Error(),
			)
		}
	}
	return err
}

func (e editerImpl) edit(ctx context.Context, m Meta) error {
	switch m.(type) {
	case *meta.EmailProducer:
		v := m.(*meta.EmailProducer)
		return e.editEmail(ctx, m, &model.DbEmail{
			ID:          v.Id,
			Content:     v.Content,
			SendTime:    v.SendTime,
			Destination: v.Destination,
		})
	case *meta.WeChatProducer:
		v := m.(*meta.WeChatProducer)
		return e.editWeChat(ctx, m, &model.DbWeChat{
			ID:       v.Id,
			Content:  v.Data,
			SendTime: v.SendTime,
			Touser:   v.Touser,
		})
	case *meta.SmsProducer:
		v := m.(*meta.SmsProducer)
		return e.editSms(ctx, m, &model.DbSms{
			ID:       v.Id,
			Content:  v.Content,
			SendTime: v.SendTime,
			Mobile:   v.Mobile,
		})
	default:
		return errors.ErrMsgTypeNotFound
	}
}

func (editerImpl) editEmail(ctx context.Context, m Meta, e *model.DbEmail) error {
	// 用于更新缓存
	em := &model.DbEmail{}
	if err := checkStatus(m.GetId(), em); err != nil {
		return err
	}

	// 修改数据
	em.Content = e.Content
	em.SendTime = e.SendTime
	if e.Destination != "" {
		em.Destination = e.Destination
	}

	tx, err := db.EmailEdit(ctx, e)
	if err != nil {
		rollback(tx)
		return err
	}

	b, err := em.Marshal()
	if err != nil {
		rollback(tx)
		return err
	}
	err = cache.PutBaseCache(ctx, m.GetId(), b)
	if err != nil {
		rollback(tx)
		return err
	}
	err = commit(tx)
	cache.PutLastestCache(ctx, m.GetId(), b)

	return err
}

func (editerImpl) editWeChat(ctx context.Context, m Meta, e *model.DbWeChat) error {
	em := &model.DbWeChat{}
	if err := checkStatus(m.GetId(), em); err != nil {
		return err
	}

	// 修改数据
	em.Content = e.Content
	em.SendTime = e.SendTime
	if e.Touser != "" {
		em.Touser = e.Touser
	}

	tx, err := db.WeChatEdit(ctx, e)
	if err != nil {
		rollback(tx)
		return err
	}

	b, err := em.Marshal()
	if err != nil {
		rollback(tx)
		return err
	}
	err = cache.PutBaseCache(ctx, m.GetId(), b)
	if err != nil {
		rollback(tx)
		return err
	}
	err = commit(tx)
	cache.PutLastestCache(ctx, m.GetId(), b)

	return err
}

func (editerImpl) editSms(ctx context.Context, m Meta, e *model.DbSms) error {
	em := &model.DbSms{}
	if err := checkStatus(m.GetId(), em); err != nil {
		return err
	}

	// 修改数据
	em.Content = e.Content
	em.SendTime = e.SendTime
	if e.Mobile != "" {
		em.Mobile = e.Mobile
	}

	tx, err := db.SmsEdit(ctx, e)
	if err != nil {
		rollback(tx)
		return err
	}

	b, err := em.Marshal()
	if err != nil {
		rollback(tx)
		return err
	}
	err = cache.PutBaseCache(ctx, m.GetId(), b)
	if err != nil {
		rollback(tx)
		return err
	}
	err = commit(tx)
	cache.PutLastestCache(ctx, m.GetId(), b)

	return err
}

func checkStatus(id string, msg Messager) error {
	b, err := cache.BaseDetail(context.Background(), id)
	// ttl := m.Delay()
	if err != nil {
		return err
	}
	err = msg.Unmarshal(b)
	if err != nil {
		return err
	}
	st := msg.GetStatus()
	if st == meta.Status_Cancel {
		return errors.ErrMsgHasCancelled
	}
	if st == meta.Status_Final {
		return errors.ErrMsgCantEdit
	}
	return nil
}