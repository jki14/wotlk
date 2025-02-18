import { Spec } from '../proto/common.js';
import { CustomRotation, CustomSpell } from '../proto/common.js';
import { EventID, TypedEvent } from '../typed_event.js';
import { Player } from '../player.js';
import { BooleanPicker } from '../components/boolean_picker.js';
import { IconEnumPicker, IconEnumPickerConfig, IconEnumValueConfig } from '../components/icon_enum_picker.js';
import { ListPicker, ListPickerConfig } from '../components/list_picker.js';
import { NumberPicker } from '../components/number_picker.js';
import { getEnumValues } from '../utils.js';

import { Component } from './component.js';

export interface CustomRotationPickerConfig<SpecType extends Spec, T> {
	getValue: (player: Player<SpecType>) => CustomRotation,
	setValue: (eventID: EventID, player: Player<SpecType>, newValue: CustomRotation) => void,

	numColumns: number,
	values: Array<IconEnumValueConfig<Player<SpecType>, T>>;

	showWhen?: (player: Player<SpecType>) => boolean,
}

export class CustomRotationPicker<SpecType extends Spec, T> extends Component {
	constructor(parent: HTMLElement, modPlayer: Player<SpecType>, config: CustomRotationPickerConfig<SpecType, T>) {
		super(parent, 'custom-rotation-picker-root');

		new ListPicker<Player<SpecType>, CustomSpell, CustomSpellPicker<SpecType, T>>(this.rootElem, modPlayer, {
			extraCssClasses: [
				'custom-spells-picker',
			],
			title: 'Spell Priority',
			titleTooltip: 'Spells at the top of the list are prioritized first. Safely ignores untalented options.',
			itemLabel: 'Spell',
			changedEvent: (player: Player<SpecType>) => player.changeEmitter,
			getValue: (player: Player<SpecType>) => config.getValue(player).spells,
			setValue: (eventID: EventID, player: Player<SpecType>, newValue: Array<CustomSpell>) => {
				config.setValue(eventID, player, CustomRotation.create({
					spells: newValue,
				}));
			},
			newItem: () => CustomSpell.create(),
			copyItem: (oldItem: CustomSpell) => CustomSpell.clone(oldItem),
			newItemPicker: (parent: HTMLElement, newItem: CustomSpell, listPicker: ListPicker<Player<SpecType>, CustomSpell, CustomSpellPicker<SpecType, T>>) => new CustomSpellPicker(parent, modPlayer, newItem, config, listPicker),
			inlineMenuBar: true,
			showWhen: config.showWhen,
		});
	}
}

class CustomSpellPicker<SpecType extends Spec, T> extends Component {
	private readonly player: Player<SpecType>;
	private readonly config: CustomRotationPickerConfig<SpecType, T>;
	private readonly listPicker: ListPicker<Player<SpecType>, CustomSpell, CustomSpellPicker<SpecType, T>>;

	constructor(parent: HTMLElement, player: Player<SpecType>, modSpell: CustomSpell, config: CustomRotationPickerConfig<SpecType, T>, listPicker: ListPicker<Player<SpecType>, CustomSpell, CustomSpellPicker<SpecType, T>>) {
		super(parent, 'custom-spell-picker-root');
		this.player = player;
		this.config = config;
		this.listPicker = listPicker;

		new IconEnumPicker<CustomSpell, number>(this.rootElem, modSpell, {
			numColumns: config.numColumns,
			values: config.values.map(value => {
				if (value.showWhen) {
					const oldShowWhen = value.showWhen;
					value.showWhen = ((spell: CustomSpell) => oldShowWhen(player)) as unknown as ((player: Player<SpecType>) => boolean);
				}
				return value;
			}) as unknown as Array<IconEnumValueConfig<CustomSpell, number>>,
			equals: (a: number, b: number) => a == b,
			zeroValue: 0,
			changedEvent: (spell: CustomSpell) => player.changeEmitter,
			getValue: (spell: CustomSpell) => spell.spell,
			setValue: (eventID: EventID, spell: CustomSpell, newValue: number) => {
				spell.spell = newValue;
				this.setSpell(eventID, spell);
			},
		});
	}

	private setSpell(eventID: EventID, spell: CustomSpell) {
		const index = this.listPicker.getPickerIndex(this);
		const cr = this.config.getValue(this.player);
		cr.spells[index] = CustomSpell.clone(spell);
		this.config.setValue(eventID, this.player, cr);
	}
}
