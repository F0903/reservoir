<script lang="ts">
    let {
        label,
        value = $bindable(),
        tooltip,
    }: {
        label: string;
        value: boolean;
        tooltip?: string;
    } = $props();

    function onToggleClick() {
        value = !value;
    }
</script>

<div class="input-container">
    <input
        type="checkbox"
        class="input"
        id="{label}-toggle"
        bind:checked={value}
        title={tooltip}
        aria-checked={value}
        role="switch"
    />
    <label class="label" for="{label}-toggle" title={tooltip}>{label}</label>
    <div class="track" class:checked={value} aria-hidden="true" onclick={onToggleClick}>
        <div class="thumb"></div>
    </div>
</div>

<style>
    .thumb {
        --thumb-size: calc(var(--track-height) - (var(--track-padding) * 2));

        width: var(--thumb-size);
        height: var(--thumb-size);
        background-color: var(--secondary-600);
        border-radius: 50%;

        box-shadow: 0px 0px 10px -3px rgba(0, 0, 0, 1);

        transition:
            transform var(--pos-trans-duration) var(--pos-trans-timing),
            background-color var(--color-trans-duration) var(--color-trans-timing);
    }

    .track.checked .thumb {
        transform: translateX(
            calc(var(--track-width) - var(--thumb-size) - (var(--track-padding) * 2))
        );
        background-color: var(--tertiary-300);
    }

    .track.checked {
        background-color: var(--tertiary-500);
    }

    .track {
        --track-height: var(--toggle-track-height, 24px);
        --track-width: var(--toggle-track-width, 45px);
        --track-padding: var(--toggle-track-padding, 2px);

        --color-trans-duration: 100ms;
        --color-trans-timing: ease-in-out;
        --pos-trans-duration: 75ms;
        --pos-trans-timing: cubic-bezier(0.86, 0, 0.07, 1);

        width: var(--track-width);
        height: var(--track-height);
        background-color: var(--primary-450);
        border-radius: calc(var(--track-height) / 2);

        display: flex;
        align-items: center;
        padding: var(--track-padding);

        transition-property: background-color;
        transition-timing-function: var(--color-trans-timing);
        transition-duration: var(--color-trans-duration);

        overflow: hidden;

        cursor: pointer;
    }

    .input-container {
        position: relative;

        display: flex;
        flex-direction: row;
        align-items: center;
        justify-content: space-between;
        gap: 15px;

        width: var(--toggle-width, 100%);

        margin: 1.2rem 0px;
    }

    .label {
        font-size: 1rem;
        color: var(--secondary-500);
        letter-spacing: 0.05em;
    }

    .input {
        opacity: 0; /* Hide the default checkbox */
        position: absolute; /* Remove it from the document flow */
        inset: 0; /* Ensure it covers the same area as the track */
        z-index: -1; /* Place it behind the custom track and thumb */
    }

    .input:focus {
        border-color: var(--secondary-400);
    }

    .input:invalid {
        border-color: var(--error-border-color);
    }
</style>
