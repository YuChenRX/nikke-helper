# Pipeline 节点命名规范

本文档用于规范 MaaFramework 项目中 Pipeline 节点的命名方式，提升节点可读性、可维护性与跨模块一致性。

Pipeline 节点名是 JSON 根对象中的 key，会被 `next`、`on_error`、`target`、`anchor`、`And` / `Or` 识别条件、`pipeline_override` 等字段引用。因此，节点名应稳定、明确，并能表达节点在流程中的职责。

## 基本原则

节点名必须使用 **PascalCase**。

推荐格式为：

```text
<Domain><ActionOrObject><Role>
```

其中：

- `Domain` 表示所属功能域或模块，例如 `DailyTask`、`Shop`、`Battle`、`Base`。
- `ActionOrObject` 表示节点处理的动作、页面、对象或业务目标。
- `Role` 表示节点在流程中的功能角色，例如 `Main`、`Flow`、`Enter`、`OnPage`、`Visible`、`Available`、`Selected`、`Confirm`、`Claim` 等。
- `Detected` 仅用于少数非 UI、异常或算法信号类检测，不作为普通 UI 元素检测的默认后缀。

示例：

```text
ShopEnterExchangePage
ShopOnExchangePage
BattleQuickBattleAvailable
DailyTaskClaimMissionReward
BaseConfirmReward
```

## 禁止事项

节点名不得使用以下形式：

```text
_StartTask1
25Check
clickReward
shop_enter
Flag_In_Shop
```

具体禁止规则：

1. 不得以下划线 `_` 开头。
2. 不得以数字开头。
3. 不得使用 `snake_case`、`camelCase` 或混合分隔符。
4. 不得使用无业务语义的临时编号，例如 `Node1`、`Check2`。
5. 不得仅用过于泛化的名称，例如 `Confirm`、`Check`、`Click`。
6. 不使用 `FlagInX` 作为新节点名；页面状态应使用 `On...Page` 或 `Visible` 表达。

## Domain 命名

`Domain` 应表达节点所属的业务域或共享域。

常见 Domain 示例：

| Domain       | 用途                                         |
| ------------ | -------------------------------------------- |
| `Base`       | 全局通用节点，例如确认、关闭、返回、空白点击 |
| `Navigation` | 跨模块导航、回到首页、进入主功能区等         |
| `Shop`       | 商店相关流程                                 |
| `Battle`     | 战斗相关流程                                 |
| `DailyTask`  | 日常任务相关流程                             |
| `Event`      | 活动相关流程                                 |

同一项目内应保持共享域命名一致。例如选择 `Base` 作为通用域后，不应同时混用 `Common`、`Global` 表达同一类节点。

## 节点角色命名

### 入口节点

任务或模块入口节点使用：

```text
<Domain>Main
```

示例：

```text
ShopMain
BattleMain
DailyTaskMain
EventMain
```

入口节点通常只负责组织后继节点，不直接承担具体识别或点击动作。

### 流程节点

只负责组织后继节点、不直接识别或点击的节点，使用：

```text
<Domain><Subtask>Flow
```

示例：

```text
ShopPurchaseItemFlow
BattleRepeatStageFlow
DailyTaskClaimRewardFlow
EventLoginRewardFlow
```

适用场景：

```json
{
    "ShopPurchaseItemFlow": {
        "next": [
            "ShopPurchaseDialogVisible",
            "[JumpBack]BaseConfirmAction",
            "BaseConfirmReward"
        ]
    }
}
```

### 进入页面节点

用于点击入口并进入某个页面的节点，使用：

```text
<Domain>Enter<Page>
```

示例：

```text
ShopEnterExchangePage
BattleEnterStagePage
DailyTaskEnterMissionPage
EventEnterLoginRewardPage
```

不要省略 Domain：

```text
EnterShop
EnterBattle
EnterMission
```

省略 Domain 会使节点在全局 Pipeline 中难以区分来源和职责。

### 页面状态节点

用于判断当前是否处于某页面、某界面、某弹窗的节点，使用：

```text
<Domain>On<Page>Page
```

或：

```text
<Domain><Object>Visible
```

示例：

```text
ShopOnExchangePage
BattleOnStagePage
DailyTaskMissionPageVisible
BaseRewardDialogVisible
```

不要使用：

```text
FlagInShop
FlagInBattle
FlagInMission
```

页面状态应描述“处于哪个页面”或“哪个 UI 对象可见”，而不是描述内部标记。

### 纯检测节点

只负责识别某个元素、状态、文本、红点，不执行动作的节点，优先使用能表达业务状态的后缀。普通 UI 元素、页面文字、按钮、红点、图标等“界面上可见”的对象，默认使用 `Visible`，不要使用 `Detected` 暴露底层识别实现。

```text
<Domain><Object>Visible
<Domain><Object>Available
<Domain><Object>Claimed
<Domain><Object>Selected
<Domain><Object>Completed
<Domain><Object>Exhausted
<Domain><Object>Detected
```

示例：

```text
DailyTaskRedDotVisible
ShopPurchaseButtonVisible
BattleQuickBattleAvailable
ShopItemSelected
DailyTaskMissionClaimed
BattleScreenFreezeDetected
```

根据语义选择后缀：

| 后缀        | 含义                                                                                                     |
| ----------- | -------------------------------------------------------------------------------------------------------- |
| `Visible`   | UI 元素、页面文字、按钮、红点、图标等在界面上可见；这是普通 UI 检测的默认后缀                            |
| `Available` | 功能、按钮、次数可用                                                                                     |
| `Claimed`   | 奖励或任务已领取                                                                                         |
| `Selected`  | 选项已选中                                                                                               |
| `Completed` | 流程、任务、收集、阶段已完成                                                                             |
| `Exhausted` | 次数、资源、机会已耗尽                                                                                   |
| `Detected`  | 非 UI 的异常、状态、算法信号被检测到；仅在 `Visible` / `Available` / `Selected` 等业务后缀都不准确时使用 |

`Visible` 与 `Detected` 的取舍规则：

- 看到界面元素：使用 `Visible`，例如 `BaseRewardDialogVisible`、`DailyTaskRedDotVisible`。
- 功能是否可点、次数是否可用：使用 `Available`，例如 `BattleQuickBattleAvailable`。
- 选项是否处于选中态：使用 `Selected`，例如 `ShopItemSelected`。
- 检测异常、冻结、颜色异常、算法信号等非 UI 语义：才使用 `Detected`，例如 `BattleScreenFreezeDetected`、`ColorAnomalyDetected`。

### 点击/选择节点

执行点击、选择、领取等动作的节点，使用动词前置：

```text
<Domain>Click<Object>
<Domain>Select<Object>
<Domain>Claim<Object>
<Domain>Purchase<Object>
<Domain>Open<Object>
<Domain>Close<Object>
```

示例：

```text
BaseClickBlank
ShopPurchaseFreeItem
BattleSelectStage
DailyTaskClaimMissionReward
BaseClosePage
```

不要使用：

```text
ClickMax
PassClick
FreeRecruitClick
```

### 确认节点

确认弹窗、确认奖励、确认操作使用：

```text
<Domain>Confirm<Object>
```

示例：

```text
BaseConfirmAction
BaseConfirmReward
ShopConfirmPurchase
BattleConfirmRetreat
```

不要使用：

```text
Confirm
ActionConfirm
RewardConfirm
ConfirmEnd
```

### 滚动/滑动节点

滚动、滑动节点使用：

```text
<Domain>Scroll<Direction>
<Domain>Swipe<Object>
```

示例：

```text
BaseScrollUp
ShopScrollItemListDown
EventSwipeBanner
BattleSwipeStageListLeft
```

不要使用：

```text
ScrollUp
SlideBanner
ListSwipe
```

### 结束节点

结束节点应明确表达结束范围：

```text
<Domain>End
<Domain>EndTask
```

示例：

```text
ShopEnd
BattleEnd
BaseEndTask
```

如果项目只需要一个全局终止节点，推荐使用 `BaseEndTask`。

## 共享节点命名

共享节点是跨模块复用的节点，应使用稳定的共享 Domain 前缀，避免和业务模块节点混淆。

推荐使用 `Base` 表示通用 UI 操作：

```text
BaseConfirmReward
BaseConfirmAction
BaseClosePage
BaseClickBlank
BaseGoBack
BaseScrollUp
BaseEndTask
```

推荐使用 `Navigation` 表示跨模块导航：

```text
NavigationEnterHome
NavigationEnterMainArea
NavigationClickHomeButton
NavigationOnHomePage
NavigationMainAreaVisible
```

共享域名称应在项目内保持唯一和一致。不要同时混用 `Base`、`Common`、`Global` 表达同一类通用节点。

## 重命名检查清单

每次重命名节点后，必须检查以下引用位置：

1. 节点定义 key。
2. `next`。
3. `on_error`。
4. `[JumpBack]NodeName`。
5. `[Anchor]AnchorName` 相关节点引用。
6. `target` 中的节点名引用。
7. `anchor` 对象中的节点名引用。
8. `recognition.type = "And"` 中的 `all_of`。
9. `recognition.type = "Or"` 中的 `any_of`。
10. `pipeline_override`。
11. 项目接口或任务配置中的任务入口与 Pipeline 覆盖配置。
12. 构建产物、安装资源或镜像资源中的重复 Pipeline 文件。

## 命名示例

### 推荐

```json
{
    "ShopMain": {
        "next": [
            "ShopEnterExchangePage",
            "[JumpBack]NavigationEnterHome"
        ]
    },
    "ShopEnterExchangePage": {
        "desc": "进入兑换商店页面",
        "recognition": {
            "type": "OCR"
        },
        "action": {
            "type": "Click"
        },
        "next": [
            "ShopOnExchangePage",
            "ShopEnterExchangePage"
        ]
    },
    "ShopOnExchangePage": {
        "desc": "处于兑换商店页面",
        "recognition": {
            "type": "OCR"
        },
        "next": [
            "[JumpBack]ShopPurchaseItemFlow",
            "BaseEndTask"
        ]
    }
}
```

### 不推荐

```json
{
    "Shop": {},
    "EnterShop": {},
    "FlagInShop": {},
    "_Shop1": {},
    "25Check": {}
}
```

## 总结

Pipeline 节点命名统一采用：

```text
PascalCase + 模块域 + 功能语义 + 角色后缀
```

节点名应优先表达“节点在流程中的功能”，而不是表达底层识别或动作实现。

推荐风格：

```text
ShopEnterExchangePage
ShopOnExchangePage
BattleQuickBattleAvailable
DailyTaskMissionClaimed
BaseConfirmReward
```

避免风格：

```text
FlagInShop
ClickMax
25Check
_Shop1
Confirm
```
