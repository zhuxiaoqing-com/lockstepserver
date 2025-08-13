package game

import (
	"github.com/byebyebruce/lockstepserver/pb"
	"github.com/golang/protobuf/proto"
)

type frameData struct {
	idx  uint32
	cmds []*pb.InputData
}

func newFrameData(index uint32) *frameData {
	f := &frameData{
		idx:  index,
		cmds: make([]*pb.InputData, 0),
	}

	return f
}

type lockstep struct {
	// 这里用map和list的区别就是, 如果有input和event的话是没区别 每一帧输入都必须有,因为我需要复用客户端的Input;
	// 如果只有event的帧同步的话,那客户端没有上传操作;这一帧的数据可以直接不创建;
	// 所以最好使用map, 因为两种都可以兼容
	frames     map[uint32]*frameData
	frameCount uint32
}

func newLockstep() *lockstep {
	l := &lockstep{
		frames: make(map[uint32]*frameData),
	}

	return l
}

func (l *lockstep) reset() {
	l.frames = make(map[uint32]*frameData)
	l.frameCount = 0
}

func (l *lockstep) getFrameCount() uint32 {
	return l.frameCount
}

func (l *lockstep) pushCmd(cmd *pb.InputData) bool {
	f, ok := l.frames[l.frameCount]
	if !ok {
		f = newFrameData(l.frameCount)
		l.frames[l.frameCount] = f
	}

	if cmd.Input != nil {
		//l4g.Warn("network receive input FrameId %s , sid %s Time %s", f, *cmd.Input.Sid, time.Now().UnixMilli())
	}

	// 这里不应该丢弃 但是因为go语言不太熟悉 就先不改了;
	// 检查是否同一帧发来两次操作
	for _, v := range f.cmds {
		if v.Id == cmd.Id {
			return false
		}
	}

	f.cmds = append(f.cmds, cmd)

	return true
}

func (l *lockstep) tick(game *Game) uint32 {
	l.frameCount++
	l.onTickFrame(game)
	return l.frameCount
}

func (l *lockstep) getRangeFrames(from, to uint32) []*frameData {
	ret := make([]*frameData, 0, to-from)

	for ; from <= to && from <= l.frameCount; from++ {
		f, ok := l.frames[from]
		if !ok {
			continue
		}
		ret = append(ret, f)
	}

	return ret
}

func (l *lockstep) getFrame(idx uint32) *frameData {
	return l.frames[idx]
}

/*
*掉线后立即停止输入复用，角色进入“静止”状态 (这里go 代码比较难弄  也先不弄这个了) 应该是离线以后立即输入一个空输入;
// 如果有Input又有event的需要每帧都给服务器发操作,服务器再通过是不是有操作,客户端没有input就继承上一帧的input,

	// 防止比如玩家一直输入D键,但是因为网络等问题 服务器有一帧没有收到客户端输入,而错误的将其任务该帧客户端没有任何输入;

	// 如果是只有event那就 不需要每帧发送input, 因为服务器不需要继承客户端的上一个输入,而是直接转发就好了;
*/
func (l *lockstep) onTickFrame(game *Game) {
	// 应该即将广播上一帧 所以处理上一帧
	idx := l.frameCount - 1
	// 如果游戏是同时有 input 和 event 的
	pre := idx - 1
	var preData *frameData = nil

	if l.frames[pre] != nil {
		preData = l.frames[pre]
	}
	// 前一个帧也是空输入就不管
	if preData == nil {
		return
	}

	data := l.frames[idx]
	if data == nil {
		data = newFrameData(idx)
		l.frames[idx] = data
	}

	for i := range game.players {
		player := game.players[i]
		var find = false
		for _, v := range data.cmds {
			if *v.Id == player.id {
				find = true
			}
		}

		if !find {
			for _, v := range preData.cmds {
				// 找到了上一帧的输入
				if *v.Id == player.id {
					var result *pb.PBInput

					// 玩家在线继承上一帧， 玩家离线直接空输入
					if player.isOnline {
						result = proto.Clone(v.Input).(*pb.PBInput)
					} else {
						result = nil
					}

					serverCreateCmd := &pb.InputData{
						Id:         proto.Uint64(player.id),
						Event:      nil, // 事件不继承
						Input:      result,
						Roomseatid: proto.Int32(player.idx),
					}
					data.cmds = append(data.cmds, serverCreateCmd)
				}
			}
		}
	}

}
