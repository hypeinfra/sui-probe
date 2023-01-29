package sui

import (
	"testing"
)

func TestNodeClient_GetTotalTransactionNumber(t *testing.T) {
	node := NewNode(OfficialDevNode)
	tests := []struct {
		name     string
		node     *NodeClient
		moreThan int // we want this value to be >0 in this test
		wantErr  bool
	}{
		{"Test", node, 0, false}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := tt.node
			got, err := n.GetTotalTransactionNumber()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTotalTransactionNumber() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got <= tt.moreThan {
				t.Errorf("GetTotalTransactionNumber() got = %v, but it should be more than %v", got, tt.moreThan)
			}
		})
	}
}
