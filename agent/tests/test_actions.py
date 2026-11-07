from wechat_agent.actions import PlannerDecision, UIAction


def test_action_drops_nulls_for_irrelevant_fields():
    decision = PlannerDecision.model_validate(
        {
            "analysis": "click search",
            "action": {
                "kind": "click",
                "label": 3,
                "direction": None,
                "scroll_type": None,
                "wheel_times": None,
                "duration_seconds": None,
                "condition": None,
                "wait_timeout_seconds": None,
            },
        }
    )

    assert decision.action.kind == "click"
    assert decision.action.label == 3
    assert decision.action.direction == "down"
    assert decision.action.wheel_times == 1


def test_scroll_zero_wheel_times_is_normalized():
    action = UIAction.model_validate(
        {
            "kind": "scroll",
            "label": 4,
            "direction": "down",
            "wheel_times": 0,
        }
    )

    assert action.wheel_times == 1


def test_type_action_can_use_current_focus():
    action = UIAction.model_validate(
        {
            "kind": "type",
            "text": "target account",
            "press_enter": True,
        }
    )

    assert action.label is None
    assert action.loc is None
    assert action.text == "target account"
