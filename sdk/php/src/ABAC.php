<?php
declare(strict_types=1);

namespace Ggid\Sdk;

/**
 * ABAC trait — provides attribute-based access control methods.
 *
 * Intended to be used by GGIDClient.
 */
trait ABAC
{
    /**
     * Evaluate ABAC policy with structured conditions.
     *
     * @param string $token Access token
     * @param string $action Action to evaluate (e.g. "transfer")
     * @param string $resource Resource being accessed (e.g. "inventory")
     * @param string $subject Subject identifier (user ID)
     * @param array $conditions Array of {field, operator, value} condition objects
     * @param string|null $tenantId Override tenant ID
     * @return ABACResult
     */
    public function evaluateABAC(
        string $token,
        string $action,
        string $resource,
        string $subject,
        array $conditions = [],
        ?string $tenantId = null,
    ): ABACResult {
        $body = [
            'action' => $action,
            'resource' => $resource,
            'subject' => $subject,
        ];
        if (!empty($conditions)) {
            $body['conditions'] = $conditions;
        }
        if ($tenantId !== null) {
            $body['tenant_id'] = $tenantId;
        }
        $data = $this->request('POST', '/api/v1/policies/abac/evaluate', $body, $token);
        return ABACResult::fromArray($data);
    }

    /**
     * Full ABAC policy check with subject context.
     *
     * @param string $token Access token
     * @param string $subject Subject identifier
     * @param string $resource Resource being accessed
     * @param string $action Action to evaluate
     * @param array $context Additional context attributes
     * @return ABACResult
     */
    public function checkPolicy(
        string $token,
        string $subject,
        string $resource,
        string $action,
        array $context = [],
    ): ABACResult {
        $body = [
            'subject' => $subject,
            'resource' => $resource,
            'action' => $action,
        ];
        if (!empty($context)) {
            $body['context'] = $context;
        }
        $data = $this->request('POST', '/api/v1/policies/abac/evaluate', $body, $token);
        return ABACResult::fromArray($data);
    }
}
