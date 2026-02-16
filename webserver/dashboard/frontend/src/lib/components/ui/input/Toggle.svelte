<script lang="ts">
    let {
        label,
        value = $bindable(),
        tooltip,
        disabled = false,
    }: {
        label: string;
        value: boolean;
        tooltip?: string;
        disabled?: boolean;
    } = $props();

    function onToggleClick() {
        value = !value;
    }
</script>

<div class="toggle-wrapper">
    <label class="label" for="{label}-toggle" title={tooltip}>{label}</label>
    <div class="input-container">
        <input
            type="checkbox"
            class="input"
            id="{label}-toggle"
            bind:checked={value}
            title={tooltip}
            aria-checked={value}
            role="switch"
            {disabled}
        />
        <div
            class="track"
            class:checked={value}
            aria-hidden="true"
            onclick={onToggleClick}
            class:disabled
        >
            <div class="thumb"></div>
        </div>
        <span class="status-text">{value ? "Enabled" : "Disabled"}</span>
    </div>
</div>

<style>
    .toggle-wrapper {
        display: flex;
        flex-direction: column;
        gap: 0.4rem;
        margin: 0.75rem 0;
        width: var(--toggle-width, 100%);
    }

    .input-container {
        display: flex;
        flex-direction: row;
        align-items: center;
        gap: 1rem;
        height: 42px;
    }

    .label {
        font-size: 0.75rem;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.08em;
        color: var(--secondary-300);
        opacity: 0.8;
        padding-left: 0.2rem;
    }

    .status-text {
        font-size: 0.85rem;
        font-weight: 600;
        color: var(--text-400);
        opacity: 0.6;
    }

    .track {
        --track-height: 24px;
        --track-width: 48px;
        --track-padding: 3px;

        width: var(--track-width);
        height: var(--track-height);
        background-color: var(--primary-600);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: calc(var(--track-height) / 2);

        display: flex;
        align-items: center;
        padding: var(--track-padding);
        cursor: pointer;
        transition: all 0.2s ease;
    }

    .track.checked {
        background-color: var(--success-color);
        border-color: var(--success-border);
    }

    .thumb {
        width: calc(var(--track-height) - (var(--track-padding) * 2) - 2px);
        height: calc(var(--track-height) - (var(--track-padding) * 2) - 2px);
        background-color: #fff;
        border-radius: 50%;
        box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
        transition: transform 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
    }

    .track.checked .thumb {
        transform: translateX(calc(var(--track-width) - var(--track-height)));
    }

    .track.disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .input {
        position: absolute;
        width: 1px;
        height: 1px;
        padding: 0;
        margin: -1px;
        overflow: hidden;
        clip: rect(0, 0, 0, 0);
        white-space: nowrap;
        border-width: 0;
    }
</style>
