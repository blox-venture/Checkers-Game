package keeper

import (
	"context"
	"github.com/alice/checkers/x/checkers/rules"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/alice/checkers/x/checkers/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) RejectGame(goCtx context.Context, msg *types.MsgRejectGame) (*types.MsgRejectGameResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	storedGame, found := k.Keeper.GetStoredGame(ctx, msg.GameIndex)
	if !found {
		return nil, sdkerrors.Wrapf(types.ErrGameNotFound, "%s", msg.GameIndex)
	}

	if storedGame.Winner != rules.PieceStrings[rules.NO_PLAYER] {
		return nil, types.ErrGameFinished
	}

	// Did the player already play?
	if storedGame.Black == msg.Creator {
		if 0 < storedGame.MoveCount {
			return nil, types.ErrBlackAlreadyPlayed
		}
	} else if storedGame.Red == msg.Creator {
		if 1 < storedGame.MoveCount {
			return nil, types.ErrRedAlreadyPlayed
		}
	} else {
		return nil, sdkerrors.Wrapf(types.ErrCreatorNotPlayer, "%s", msg.Creator)
	}

	k.Keeper.MustRefundWager(ctx, &storedGame)

	// remove a game from FIFO
	systemInfo, found := k.Keeper.GetSystemInfo(ctx)
	if !found {
		panic("SystemInfo not found")
	}
	k.Keeper.RemoveFromFifo(ctx, &storedGame, &systemInfo)

	// remove the game and update the system info
	k.Keeper.RemoveStoredGame(ctx, msg.GameIndex)
	k.Keeper.SetSystemInfo(ctx, systemInfo)

	// make sure to not return more than consumed on game rejection
	refund := uint64(types.RejectGameRefundGas)
	if consumed := ctx.GasMeter().GasConsumed(); consumed < refund {
		refund = consumed
	}
	ctx.GasMeter().RefundGas(refund, "Reject game")

	// emit the relevant event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(types.GameRejectedEventType,
			sdk.NewAttribute(types.GameRejectedEventCreator, msg.Creator),
			sdk.NewAttribute(types.GameRejectedEventGameIndex, msg.GameIndex),
		),
	)

	return &types.MsgRejectGameResponse{}, nil
}
