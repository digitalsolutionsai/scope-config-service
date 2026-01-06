package vn.dsai.scopeconfig;

import vn.dsai.config.v1.ConfigIdentifier;
import vn.dsai.config.v1.Scope;

/**
 * Options for retrieving a configuration value.
 */
public class GetValueOptions {
    
    /**
     * Use default value from template if config value is not set.
     */
    private final boolean useDefault;
    
    /**
     * Traverse parent scopes to find the value.
     * Hierarchy: STORE → PROJECT → SYSTEM, USER → SYSTEM
     */
    private final boolean inherit;
    
    private GetValueOptions(Builder builder) {
        this.useDefault = builder.useDefault;
        this.inherit = builder.inherit;
    }
    
    public boolean isUseDefault() {
        return useDefault;
    }
    
    public boolean isInherit() {
        return inherit;
    }
    
    /**
     * Creates a new builder.
     */
    public static Builder builder() {
        return new Builder();
    }
    
    /**
     * Creates default options (no inheritance, no defaults).
     */
    public static GetValueOptions defaults() {
        return new Builder().build();
    }
    
    /**
     * Creates options with both inheritance and default values enabled.
     */
    public static GetValueOptions withInheritanceAndDefaults() {
        return new Builder()
            .useDefault(true)
            .inherit(true)
            .build();
    }
    
    /**
     * Builder for GetValueOptions.
     */
    public static class Builder {
        private boolean useDefault = false;
        private boolean inherit = false;
        
        private Builder() {}
        
        /**
         * Enable using default values from template.
         */
        public Builder useDefault(boolean useDefault) {
            this.useDefault = useDefault;
            return this;
        }
        
        /**
         * Enable scope inheritance.
         */
        public Builder inherit(boolean inherit) {
            this.inherit = inherit;
            return this;
        }
        
        public GetValueOptions build() {
            return new GetValueOptions(this);
        }
    }
}
