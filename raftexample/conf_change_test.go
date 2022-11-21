package raftexample

// conf change example
//confChange := raftpb.ConfChangeV2{
//	Transition: 0, // 0 for automatic decided using consensus or not
//	Changes: []raftpb.ConfChangeSingle{raftpb.ConfChangeSingle{
//		Type:   raftpb.ConfChangeAddNode,
//		NodeID: 4,
//	}},
//	Context: context.TODO(),
//}
// Node.ProposeConfChange(context.TODO(), confChange)
