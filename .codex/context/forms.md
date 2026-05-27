# Form and dynamic-list rules

## General form rules

- Формы не режутся частично.
- Non-native формы не являются источником `RefDrivenInclusion`.
- Если форма не должна переноситься — удаляется целиком.

## Native form cleanup boundary

Для `Native` forms:
- aggressive dynamic-list cleanup запрещен;
- aggressive command cleanup запрещен;
- нельзя менять standard command flags (`Create`/`Copy`/`Delete`/`Change` и аналоги) generic cleanup-ом;
- нельзя удалять native form command references только потому, что generic resolver не доказал их existence;
- нельзя удалять `MainTable`;
- нельзя удалять custom/manual-query dynamic-list fields только потому, что они отсутствуют в metadata owner;
- нельзя массово удалять `DataPath` через generic missing-field cleanup;
- cleanup должен удалять только доказанно dangling/non-native references.

Temporary safety mode:
- при `enableNativeFormCleanup=false` Native forms не проходят generic form cleanup pipeline;
- в safety mode допускается только узкий deterministic cleanup явно невалидных `ExcludedCommand`, которые дают platform load errors;
- это временный safety switch/TODO, пока не будет доказан безопасный минимальный cleanup для Native forms.

Aggressive cleanup допускается только для non-Native forms (`Adopted` / `AdoptedStub*`).

## Target-based Adopted forms

Для object Adopted-форм canonical borrowing строится из `target.xml_dump`, а не из source form composition.

Условия переноса:
- в `упо_SearchResult` есть form-level code insertion для конкретной формы;
- в target есть соответствующая форма.

Поиск target-формы:
- сначала по identifier/uuid формы;
- затем по полному metadata name (`<Owner>.Form.<FormName>`).

Если target match не найден:
- form preserve снимается;
- форма не должна оставаться в non-Native составе как source-derived partial stub.

Canonical borrowed pattern:
- top-level `Forms/<FormName>.xml` получает `ObjectBelonging=Adopted`;
- `ExtendedConfigurationObject` указывает на uuid target-формы;
- добавляется `InternalInfo/PropertyState(Form=Extended)`;
- `Ext/Form.xml` строится из target form composition и затем детерминированно переписывается в configurator-shaped borrowed form.

Configurator-shaped borrowed form:
- сохраняет target visual/item composition;
- обнуляет `CommandName` в `0`;
- удаляет `Commands`;
- удаляет `CommandInterface`;
- удаляет `Events`;
- удаляет plain `DataPath`;
- очищает `Attributes`;
- переписывает `xr:DataPath` в internal target-based refs.

Cleanup boundary для borrowed Adopted-форм:
- после target-based rewrite borrowed object form не должен проходить generic partial form cleanup;
- `cleanupExcludedReferences` не должен вырезать из borrowed target-form subtree по обычным adopted/excluded rules;
- это отдельный deterministic target-borrow path, а не частичная чистка source-derived формы.

Deterministic adopted cleanup после rewrite:
- cleanup правил для borrowed `Adopted`-форм выводится из configurator-borrowed эталонов в `input/comerr`, а не из source form composition;
- cleanup применяется общими правилами для всех borrowed `Adopted`-форм, а не per-form hardcode-списками;
- из borrowed `Adopted`-форм удаляются `ExcludedCommand`, которые системно отсутствуют в configurator-borrowed pattern и дают load errors.

Подтвержденные borrowed `Adopted` cleanup rules:
- удаляются `CreateFolder`, `Pickup`, `Choose`, `Refresh`;
- удаляются `ClearTableMarksAppearance`, `FindByCurrentValue`, `SearchEverywhere`, `SearchHistory`;
- сохраняется distinction между form-level и nested `CommandSet`;
- form-level `ExcludedCommand` может сохраняться, если он подтвержден configurator-borrowed эталоном;
- `SetDeletionMark` допустим только на root `CommandSet`, если он подтвержден borrowed эталоном;
- nested/table `CommandSet` не должен сохранять `SetDeletionMark`.

## AdoptedStubExt(Form)

Используется для form-driven target objects.

Field contract включает:
- `MainTable`
- `DataPath`
- `Field`
- `RowPictureDataPath`
- related dynamic-list references

## Dynamic list validation

`validate dynamic list contracts`:
- выполняется после XML rewrite;
- проверяет только form-driven contracts;
- нужен для non-Native target objects;
- не является full platform validation.

## Field sources

Field считается доступным через:
- `StandardAttributes`
- `Attribute`
- `Dimension`
- `Resource`
- `Measure`

## Standard attribute aliases

Минимальный набор:
- `Recorder -> Регистратор`
- `Period -> Период`
- `LineNumber -> НомерСтроки`
- `Active -> Активность`

## Dynamic-list virtual fields

Стандартные/virtual list fields должны сохраняться даже если они не найдены через обычный metadata lookup.

Минимальный allowlist:
- `Ссылка`
- `Наименование`
- `Код`
- `ПометкаУдаления`
- `Представление`

## Common failure patterns

- broken `DataPath`
- removed `CommonAttribute`
- dangling metadata-command
- partial form cleanup
- aggressive cleanup applied to Native forms
